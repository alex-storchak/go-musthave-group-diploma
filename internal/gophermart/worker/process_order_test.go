package worker

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	repoMocks "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/mocks"
	accrualconfig "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/worker/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewProcessOrder(t *testing.T) {
	var db *gorm.DB

	cfg := &config.Config{
		Accrual: &accrualconfig.Config{
			RetryAfterDefault: accrualconfig.DefaultRetryAfter,
			RequestTimeout:    accrualconfig.DefaultRequestTimeout,
			Addr:              accrualconfig.DefaultAddr,
		},
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)

	mockRepo := repoMocks.NewMockRepository(t)
	po, err := NewProcessOrder(db, cfg, logger, mockRepo)

	assert.NilError(t, err)
	assert.Assert(t, po != nil)
	assert.Equal(t, 0, len(po.ordersCh)) // Было len(po.orders)
	assert.Assert(t, po.store != nil)
	assert.Assert(t, po.accrual != nil)
	assert.Assert(t, po.store == mockRepo)
	assert.Assert(t, len(po.processingIDs) == 0) // Проверяем инициализацию
}

func TestGetOrders(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)

	mockRepo := repoMocks.NewMockRepository(t)

	expectedOrders := []models.OrderProcess{{Number: "789"}}
	mockRepo.EXPECT().
		GetNewOrders(mock.Anything, mock.Anything).
		Return(expectedOrders, nil)

	po := &ProcessOrder{
		store:         mockRepo,
		logger:        logger,
		ordersCh:      make(chan models.OrderProcess, ordersBufferSize),
		processingIDs: make(map[string]struct{}),
		mu:            &sync.RWMutex{},
	}

	ctx := t.Context()
	po.GetOrders(ctx)

	select {
	case order := <-po.ordersCh:
		assert.Equal(t, "789", order.Number)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected order in channel, but got nothing")
	}

	po.mu.Lock()
	_, exists := po.processingIDs["789"]
	po.mu.Unlock()
	assert.Assert(t, exists)
}

func TestGetOrders_RepoError(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)

	mockRepo := repoMocks.NewMockRepository(t)

	mockRepo.EXPECT().
		GetNewOrders(mock.Anything, mock.Anything).
		Return(nil, myerrors.ErrRepoError)

	po := &ProcessOrder{
		store:         mockRepo,
		logger:        logger,
		ordersCh:      make(chan models.OrderProcess, ordersBufferSize),
		processingIDs: make(map[string]struct{}),
		mu:            &sync.RWMutex{},
	}

	ctx := t.Context()
	po.GetOrders(ctx)

	select {
	case <-po.ordersCh:
		t.Fatal("unexpected order in channel")
	default:
		// Канал пуст — ожидаемое поведение
	}

	po.mu.Lock()
	size := len(po.processingIDs)
	po.mu.Unlock()
	assert.Equal(t, 0, size)
}

func TestDoneProcessing(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}
	accrualResp := &models.AccrualResponse{
		Number:  "123",
		Status:  models.AccrualProcessed,
		Accrual: 100,
	}

	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(accrualResp, nil)

	mockRepo.EXPECT().
		UpdateOrderProcessed(mock.Anything, accrualResp).
		Return(nil)

	po := &ProcessOrder{
		store:         mockRepo,
		accrual:       mockAccrual,
		ordersCh:      make(chan models.OrderProcess, 1),
		processingIDs: make(map[string]struct{}),
		mu:            &sync.RWMutex{},
		logger:        logger,
	}

	// Отправляем заказ в канал
	po.ordersCh <- models.OrderProcess{Number: "123"}

	done := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		po.doneProcessing(t.Context(), done, errCh)
	}()

	// Ждём, пока воркер запустится
	select {
	case <-done:
		// Воркер завершил работу — OK
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out: doneProcessing did not finish")
	}
}

func TestStartAccrualWorker_Success(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}
	accrualResp := &models.AccrualResponse{
		Number:  "123",
		Status:  models.AccrualProcessed,
		Accrual: 100,
	}

	// Настраиваем мок accrual.Get
	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(accrualResp, nil)

	// Настраиваем мок repository.UpdateOrderProcessed
	mockRepo.EXPECT().
		UpdateOrderProcessed(mock.Anything, accrualResp).
		Return(nil)

	po := &ProcessOrder{
		store:   mockRepo,
		accrual: mockAccrual,
		logger:  logger,
		mu:      &sync.RWMutex{},
	}

	done := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		po.startAccrualWorker(t.Context(), &order, done, errCh)
	}()

	select {
	case <-done:
		// Успех — ошибок нет
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartAccrualWorker_AccrualError(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}

	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(nil, myerrors.ErrAccrual)

	po := &ProcessOrder{
		store:   mockRepo,
		accrual: mockAccrual,
		logger:  logger,
		mu:      &sync.RWMutex{},
	}

	done := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		po.startAccrualWorker(t.Context(), &order, done, errCh)
	}()

	select {
	case <-done:
		t.Fatal("expected error, but worker finished without error")
	case err := <-errCh:
		assert.Assert(t, strings.Contains(err.Error(), "error accrual"), "error should contain 'accrual error'")
	}
}

func TestStartAccrualWorker_InvalidStatus(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}
	accrualResp := &models.AccrualResponse{
		Number: "123",
		Status: models.AccrualInvalid,
	}

	// 1. Настраиваем мок accrual.Get — возвращаем ответ с Invalid-статусом
	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(accrualResp, nil)

	// 2. Настраиваем мок repository.UpdateOrderInvalid — ожидаем вызов с accrualResp
	mockRepo.EXPECT().
		UpdateOrderInvalid(mock.Anything, accrualResp).
		Return(nil) // успешное выполнение

	po := &ProcessOrder{
		store:   mockRepo,
		accrual: mockAccrual,
		logger:  logger,
		mu:      &sync.RWMutex{},
	}

	done := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	// 3. Запускаем worker в горутине
	go func() {
		po.startAccrualWorker(t.Context(), &order, done, errCh)
	}()

	// 4. Ждём либо сигнала завершения, либо ошибки
	select {
	case <-done:
		// Ожидаемое поведение: worker завершил работу без ошибок
		// Проверяем, что UpdateOrderInvalid был вызван (мок сам проверит это при cleanup)
	case err := <-errCh:
		t.Fatalf("unexpected error in errCh: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out: worker did not finish")
	}
}
