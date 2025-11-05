package worker

import (
	"context"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	accrualconfig "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/worker/mocks"
	"github.com/stretchr/testify/mock"
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

	logger, _ := zap.NewDevelopment()

	mockRepo := mocks.NewMockRepository(t)
	po, err := NewProcessOrder(db, cfg, logger, mockRepo)

	assert.NilError(t, err)
	assert.Assert(t, po != nil)
	assert.Equal(t, 0, len(po.orders))
	assert.Assert(t, po.mu != nil)
	assert.Assert(t, po.store != nil)
	assert.Assert(t, po.accrual != nil)

	assert.Assert(t, po.store == mockRepo)
}

func TestGetOrders(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockRepo := mocks.NewMockRepository(t)

	expectedOrders := []models.OrderProcess{{Number: "789"}}
	mockRepo.EXPECT().
		GetNewOrders(mock.Anything, mock.Anything).
		Return(expectedOrders, nil)

	po := &ProcessOrder{
		store:  mockRepo,
		logger: logger,
		orders: make([]models.OrderProcess, 0, ordersBufferSize),
		mu:     &sync.RWMutex{},
	}

	ctx := context.Background()
	po.GetOrders(ctx)

	assert.Equal(t, 1, len(po.orders))
	assert.Equal(t, "789", po.orders[0].Number)
}

func TestGetOrders_RepoError(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockRepo := mocks.NewMockRepository(t)

	mockRepo.EXPECT().
		GetNewOrders(mock.Anything, mock.Anything).
		Return(nil, errors.New("repo error"))

	po := &ProcessOrder{
		store:  mockRepo,
		logger: logger,
		orders: make([]models.OrderProcess, 0, ordersBufferSize),
		mu:     &sync.RWMutex{},
	}

	ctx := context.Background()
	po.GetOrders(ctx)

	// Проверяем, что orders остались пустыми
	assert.Equal(t, 0, len(po.orders))
}

func TestFindUnprocessedOrder(t *testing.T) {
	po := &ProcessOrder{
		mu: &sync.RWMutex{},
		orders: []models.OrderProcess{
			{Number: "123"},
			{Number: "456"},
		},
	}

	// Первый вызов
	order, found := po.FindUnprocessedOrder()
	assert.Check(t, found)
	assert.Equal(t, "123", order.Number)

	// Второй вызов
	order, found = po.FindUnprocessedOrder()
	assert.Check(t, found)
	assert.Equal(t, "456", order.Number)

	// Третий вызов — заказов нет
	order, found = po.FindUnprocessedOrder()
	assert.Check(t, !found)
	assert.Assert(t, order == nil)
}

func TestStartAccrualWorker_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := mocks.NewMockRepository(t)
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
		po.startAccrualWorker(context.Background(), &order, done, errCh)
	}()

	select {
	case <-done:
		// Успех — ошибок нет
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartAccrualWorker_AccrualError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := mocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}

	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(nil, errors.New("accrual error"))

	po := &ProcessOrder{
		store:   mockRepo,
		accrual: mockAccrual,
		logger:  logger,
		mu:      &sync.RWMutex{},
	}

	done := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		po.startAccrualWorker(context.Background(), &order, done, errCh)
	}()

	select {
	case <-done:
		t.Fatal("expected error, but worker finished without error")
	case err := <-errCh:
		assert.Assert(t, strings.Contains(err.Error(), "accrual error"), "error should contain 'accrual error'")
	}
}

func TestStartAccrualWorker_InvalidStatus(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := mocks.NewMockRepository(t)
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
		po.startAccrualWorker(context.Background(), &order, done, errCh)
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
