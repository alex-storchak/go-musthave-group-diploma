package worker

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/worker/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"strconv"
	"testing"
	"time"
)

func testAccrualCfg() config.Accrual {
	return config.Accrual{
		WorkerCount:             2,
		JobChanSize:             16,
		BatchSize:               10,
		PollInterval:            10 * time.Millisecond,
		RulesCacheTTL:           0,
		StuckOrderTimeout:       50 * time.Millisecond,
		StuckOrderCheckInterval: 15 * time.Millisecond,
		StuckOrderBatchLimit:    100,
	}
}

func TestAccrualPool_Start_DispatchesBatchAndProcessesOrders(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 700*time.Millisecond)
	defer cancel()

	orderRepo := new(mocks.MockProcessOrdersRepository)
	rulesProv := new(mocks.MockRulesProvider)
	proc := new(mocks.MockOrderProcessor)

	cfg := testAccrualCfg()
	logger := zap.NewNop()

	orders := []model.Order{
		{Number: "1", Status: model.StatusProcessing},
		{Number: "2", Status: model.StatusProcessing},
	}
	rules := []model.RewardRule{
		{Match: "Good 1", Reward: 10, RewardType: model.Percent},
	}

	// Первый вызов — пачка, далее — пусто
	orderRepo.EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return(orders, nil).
		Once()
	orderRepo.EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return([]model.Order{}, nil).
		Maybe()
	orderRepo.EXPECT().
		ResetStuckOrders(mock.Anything, cfg.StuckOrderTimeout, cfg.StuckOrderBatchLimit).
		Return(0, nil).
		Maybe()

	// Rules для каждого заказа
	rulesProv.EXPECT().
		All(mock.Anything).
		Return(rules, nil).
		Times(len(orders))

	// ProcessOrder + UpdateStatus для каждого заказа
	proc.EXPECT().
		ProcessOrder(mock.AnythingOfType("*model.Order"), rules).
		Return().
		Times(len(orders))
	orderRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.AnythingOfType("*model.Order")).
		Return(nil).
		Times(len(orders))

	rulesProv.EXPECT().
		Close().
		Return().
		Once()

	pool := NewAccrualPool(proc, orderRepo, rulesProv, &cfg, logger, nil)
	pool.Start(ctx)

	time.Sleep(150 * time.Millisecond)
	pool.Close()
}

func TestAccrualPool_Worker_RulesAllErrorStillUpdatesStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	orderRepo := mocks.NewMockProcessOrdersRepository(t)
	rulesProv := mocks.NewMockRulesProvider(t)
	proc := mocks.NewMockOrderProcessor(t)

	cfg := testAccrualCfg()
	cfg.WorkerCount = 1
	cfg.JobChanSize = 4
	cfg.PollInterval = 50 * time.Millisecond
	logger := zap.NewNop()

	orders := []model.Order{{Number: "123", Status: model.StatusProcessing}}

	orderRepo.EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return(orders, nil).
		Once()

	orderRepo.EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return([]model.Order{}, nil).
		Maybe()

	orderRepo.EXPECT().
		ResetStuckOrders(mock.Anything, cfg.StuckOrderTimeout, cfg.StuckOrderBatchLimit).
		Return(0, nil).
		Maybe()

	rulesProv.EXPECT().
		All(mock.Anything).
		Return(nil, assert.AnError).
		Once()

	// ProcessOrder вызываться не должен
	proc.AssertNotCalled(t, "ProcessOrder", mock.Anything, mock.Anything)

	orderRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.AnythingOfType("*model.Order")).
		Return(nil).
		Once()

	rulesProv.EXPECT().
		Close().
		Return().
		Once()

	pool := NewAccrualPool(proc, orderRepo, rulesProv, &cfg, logger, nil)
	pool.Start(ctx)

	time.Sleep(150 * time.Millisecond)

	cancel()
	pool.Close()
}

func TestAccrualPool_ResetStuckOrders_DeterministicTicks(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	orderRepo := new(mocks.MockProcessOrdersRepository)
	rulesProv := new(mocks.MockRulesProvider)
	proc := new(mocks.MockOrderProcessor)

	cfg := testAccrualCfg()
	logger := zap.NewNop()

	ft := ticker.NewFakeTicker()
	tf := &ticker.FakeTickerFactory{T: ft}

	const N = 10
	called := make(chan struct{}, N)

	orderRepo.EXPECT().
		ResetStuckOrders(mock.Anything, cfg.StuckOrderTimeout, cfg.StuckOrderBatchLimit).
		RunAndReturn(func(ctx context.Context, timeout time.Duration, batchLimit int) (int, error) {
			called <- struct{}{}
			return 0, nil
		}).
		Times(N)

	orderRepo.EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return([]model.Order{}, nil).
		Maybe()

	proc.EXPECT().
		ProcessOrder(mock.Anything, mock.AnythingOfType("[]model.RewardRule")).
		Return().
		Maybe()

	rulesProv.EXPECT().
		Close().
		Return().
		Once()

	pool := NewAccrualPool(proc, orderRepo, rulesProv, &cfg, logger, tf)
	pool.Start(ctx)

	// Посылаем ровно N тиков
	for i := 0; i < N; i++ {
		ft.Ch <- time.Now()
	}

	// Дожидаемся N вызовов
	for i := 0; i < N; i++ {
		select {
		case <-called: // ok
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for ResetStuckOrders call #%d", i+1)
		}
	}

	cancel()
	pool.Close()
}

func TestAccrualPool_Close_WaitsForAllAndClosesProvider(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	orderRepo := new(mocks.MockProcessOrdersRepository)
	rulesProv := new(mocks.MockRulesProvider)
	proc := new(mocks.MockOrderProcessor)

	cfg := testAccrualCfg()
	cfg.PollInterval = 10 * time.Millisecond
	logger := zap.NewNop()

	// Разрешаем сколько угодно вызовов:
	orderRepo.EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return([]model.Order{}, nil).
		Maybe()
	orderRepo.EXPECT().
		ResetStuckOrders(mock.Anything, cfg.StuckOrderTimeout, cfg.StuckOrderBatchLimit).
		Return(0, nil).
		Maybe()

	rulesProv.EXPECT().
		Close().
		Return().
		Once()

	pool := NewAccrualPool(proc, orderRepo, rulesProv, &cfg, logger, nil)
	pool.Start(ctx)
	pool.Close()
}

func TestAccrualPool_Integration_ConcurrentProcessing_Expecter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	orderRepo := new(mocks.MockProcessOrdersRepository)
	rulesProv := new(mocks.MockRulesProvider)
	proc := new(mocks.MockOrderProcessor)

	cfg := testAccrualCfg()
	cfg.WorkerCount = 4
	cfg.BatchSize = 8
	cfg.JobChanSize = 8
	cfg.PollInterval = 5 * time.Millisecond
	logger := zap.NewNop()

	orders := make([]model.Order, 0, cfg.BatchSize)
	for i := 0; i < cfg.BatchSize; i++ {
		orders = append(orders, model.Order{
			Number: strconv.Itoa(i),
			Status: model.StatusProcessing,
		})
	}

	rules := []model.RewardRule{{Match: "Good 1", Reward: 1, RewardType: model.Points}}

	orderRepo.
		EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return(orders, nil).
		Once()
	orderRepo.
		EXPECT().
		GetBatchForProcessing(mock.Anything, cfg.BatchSize).
		Return([]model.Order{}, nil).
		Maybe()

	rulesProv.
		EXPECT().
		All(mock.Anything).
		Return(rules, nil).
		Times(len(orders))

	proc.
		EXPECT().
		ProcessOrder(mock.AnythingOfType("*model.Order"), rules).
		Return().
		Times(len(orders))

	orderRepo.
		EXPECT().
		UpdateStatus(mock.Anything, mock.AnythingOfType("*model.Order")).
		Return(nil).
		Times(len(orders))

	orderRepo.
		EXPECT().
		ResetStuckOrders(mock.Anything, cfg.StuckOrderTimeout, cfg.StuckOrderBatchLimit).
		Return(0, nil).
		Maybe()

	rulesProv.
		EXPECT().
		Close().
		Return().
		Once()

	pool := NewAccrualPool(proc, orderRepo, rulesProv, &cfg, logger, nil)
	pool.Start(ctx)
	time.Sleep(250 * time.Millisecond)
	pool.Close()
}
