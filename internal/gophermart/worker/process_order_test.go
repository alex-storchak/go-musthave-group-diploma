package worker

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	repoMocks "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual"
	accrualconfig "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/worker/mocks"
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
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
	assert.Equal(t, 0, len(po.ordersCh))
	assert.Assert(t, po.store != nil)
	assert.Assert(t, po.accrual != nil)
	assert.Assert(t, po.store == mockRepo)
}

func TestGetOrders(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)

	mockRepo := repoMocks.NewMockRepository(t)

	expectedOrders := []models.OrderProcess{{Number: "789"}}
	mockRepo.EXPECT().
		GetNewOrders(mock.Anything).
		Return(expectedOrders, nil)

	po := &ProcessOrder{
		store:    mockRepo,
		logger:   logger,
		ordersCh: make(chan models.OrderProcess, ordersBufferSize),
		mu:       &sync.Mutex{},
	}

	ctx := t.Context()
	po.GetOrders(ctx)

	select {
	case order := <-po.ordersCh:
		assert.Equal(t, "789", order.Number)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected order in channel, but got nothing")
	}
}

func TestGetOrders_RepoError(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)

	mockRepo := repoMocks.NewMockRepository(t)

	mockRepo.EXPECT().
		GetNewOrders(mock.Anything).
		Return(nil, tassert.AnError)

	po := &ProcessOrder{
		store:    mockRepo,
		logger:   logger,
		ordersCh: make(chan models.OrderProcess, ordersBufferSize),
		mu:       &sync.Mutex{},
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
	size := len(po.ordersCh)
	po.mu.Unlock()
	assert.Equal(t, 0, size)
}

func TestWorker_Success(t *testing.T) {
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
		store:    mockRepo,
		accrual:  mockAccrual,
		ordersCh: make(chan models.OrderProcess, 1),
		mu:       &sync.Mutex{},
		logger:   logger,
	}

	po.ordersCh <- models.OrderProcess{Number: "123"}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		po.worker(ctx, 1)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
}

func TestWorker_AccrualError(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}

	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(nil, myerrors.ErrAccrual)

	po := &ProcessOrder{
		store:    mockRepo,
		accrual:  mockAccrual,
		ordersCh: make(chan models.OrderProcess, ordersBufferSize),
		logger:   logger,
		mu:       &sync.Mutex{},
	}

	po.ordersCh <- models.OrderProcess{Number: "123"}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		po.worker(ctx, 1)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	mockRepo.AssertNotCalled(t, "UpdateOrderProcessed", mock.Anything, mock.Anything)
}

func TestWorker_InvalidStatus(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}
	accrualResp := &models.AccrualResponse{
		Number: "123",
		Status: models.AccrualInvalid,
	}

	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(accrualResp, nil)

	mockRepo.EXPECT().
		UpdateOrderInvalid(mock.Anything, accrualResp).
		Return(nil). // успешное выполнение
		Once()

	po := &ProcessOrder{
		store:    mockRepo,
		accrual:  mockAccrual,
		logger:   logger,
		ordersCh: make(chan models.OrderProcess, ordersBufferSize),
		mu:       &sync.Mutex{},
	}

	po.ordersCh <- models.OrderProcess{Number: "123"}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		po.worker(ctx, 1)
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()
}

func TestWorker_TooManyRequests(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create zap logger: %v", err)
	mockRepo := repoMocks.NewMockRepository(t)
	mockAccrual := mocks.NewMockAccrual(t)

	order := models.OrderProcess{Number: "123"}

	mockAccrual.EXPECT().
		Get(mock.Anything, order).
		Return(nil, accrual.NewRetryAfterError(1*time.Second))

	po := &ProcessOrder{
		store:    mockRepo,
		accrual:  mockAccrual,
		logger:   logger,
		ordersCh: make(chan models.OrderProcess, ordersBufferSize),
		mu:       &sync.Mutex{},
	}

	po.ordersCh <- models.OrderProcess{Number: "123"}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		po.worker(ctx, 1)
	}()

	time.Sleep(100 * time.Millisecond)

	tassert.True(t, po.shouldRetryAfter())

	cancel()

	mockRepo.AssertNotCalled(t, "UpdateOrderInvalid", mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "UpdateOrderProcessed", mock.Anything, mock.Anything)
}
