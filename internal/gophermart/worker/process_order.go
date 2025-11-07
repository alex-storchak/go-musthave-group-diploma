package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sync"
	"time"
)

const (
	maxConcurrent           = 10
	ordersBufferSize        = 50
	getOrdersTickerDuration = 5 * time.Second
)

type Accrual interface {
	Get(ctx context.Context, order models.OrderProcess) (*models.AccrualResponse, error)
}

type ProcessOrder struct {
	store      repository.Repository
	accrual    Accrual
	logger     *zap.Logger
	ordersCh   chan models.OrderProcess
	mu         *sync.Mutex
	retryUntil time.Time
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

func NewProcessOrder(conn *gorm.DB, cfg *config.Config, l *zap.Logger, repoOpt ...repository.Repository) (*ProcessOrder, error) {
	var store repository.Repository
	var err error
	if len(repoOpt) > 0 && repoOpt[0] != nil {
		store = repoOpt[0]
	} else {
		store, err = NewRepository(conn)
		if err != nil {
			return nil, fmt.Errorf("no init repository: %w", err)
		}
	}

	acc := accrual.New(cfg.Accrual)

	return &ProcessOrder{
		store:    store,
		accrual:  acc,
		logger:   l,
		ordersCh: make(chan models.OrderProcess, ordersBufferSize),
		mu:       &sync.Mutex{},
	}, nil
}

func NewRepository(conn *gorm.DB) (repository.Repository, error) {
	repo, err := pg.NewStore(conn)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}
	return repo, nil
}

func (f *ProcessOrder) StartProcessOrder(ctx context.Context) {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.RunGetOrders(ctx)
	}()

	f.startWorkers(ctx)

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.stuckOrdersWorker(ctx)
	}()
}

func (f *ProcessOrder) startWorkers(ctx context.Context) {
	for i := 0; i < maxConcurrent; i++ {
		workerID := i
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			f.worker(ctx, workerID)
		}()
	}
}

func (f *ProcessOrder) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Пауза перед обработкой заказа
			if ok := f.waitRetry(ctx, workerID); !ok {
				return
			}

			select {
			case <-ctx.Done():
				return
			case order, ok := <-f.ordersCh:
				if !ok {
					return
				}
				f.processOrder(ctx, order)
			}
		}
	}
}

func (f *ProcessOrder) getRetryAfterUnsafe() time.Duration {
	if f.retryUntil.IsZero() {
		return 0
	}

	now := time.Now()
	if now.After(f.retryUntil) {
		return 0
	}
	return f.retryUntil.Sub(now)
}

func (f *ProcessOrder) shouldRetryAfter() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.getRetryAfterUnsafe()
	return d > 0
}

func (f *ProcessOrder) setRetryAfter(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if d <= 0 {
		f.retryUntil = time.Time{}
		return
	}

	newUntil := time.Now().Add(d)
	if newUntil.Before(f.retryUntil) {
		return
	}
	f.retryUntil = newUntil
}

func (f *ProcessOrder) waitRetry(ctx context.Context, workerID int) bool {
	f.mu.Lock()
	d := f.getRetryAfterUnsafe()

	if d <= 0 {
		f.mu.Unlock()
		return true
	}

	timer := time.NewTimer(d)
	f.mu.Unlock()

	f.logger.Debug("worker paused due to retry-after",
		zap.Duration("retry_after", d),
		zap.Int("worker_id", workerID))

	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	if err := ctx.Err(); err != nil {
		return false
	}

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (f *ProcessOrder) processOrder(ctx context.Context, order models.OrderProcess) {
	// Получение данных начисления
	accrualResponse, err := f.accrual.Get(ctx, order)
	if err != nil {
		var rae *accrual.RetryAfterError
		if errors.As(err, &rae) {
			f.setRetryAfter(rae.Duration)
			f.logger.Info("accrual rate limit hit, setting retry-after", zap.Duration("after", rae.Duration))
			return
		}
		f.logger.Error("failed to get accrual", zap.Error(err))
		return
	}

	// Обработка статусов
	switch accrualResponse.Status {
	case models.AccrualRegistered, models.AccrualProcessing:
	case models.AccrualInvalid:
		if err = f.store.UpdateOrderInvalid(ctx, accrualResponse); err != nil {
			f.logger.Error("failed to update invalid order", zap.Error(err))
		}
	case models.AccrualProcessed:
		if err := f.store.UpdateOrderProcessed(ctx, accrualResponse); err != nil {
			f.logger.Error("failed to update processed order",
				zap.String("order_number", accrualResponse.Number),
				zap.Error(err))
		} else {
			f.logger.Info("order processed successfully",
				zap.String("order_number", accrualResponse.Number),
				zap.Float64("accrual", float64(accrualResponse.Accrual)))
		}
	default:
		f.logger.Warn("unknown accrual status",
			zap.String("status", string(accrualResponse.Status)),
			zap.String("order_number", accrualResponse.Number))
	}
}

func (f *ProcessOrder) RunGetOrders(ctx context.Context) {
	ticker := time.NewTicker(getOrdersTickerDuration)
	defer ticker.Stop()

	f.logger.Info("start process order")

	f.GetOrders(ctx)

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("stop process order")
			return
		case <-ticker.C:
			f.GetOrders(ctx)
		}
	}
}

func (f *ProcessOrder) GetOrders(ctx context.Context) {
	if f.shouldRetryAfter() {
		f.logger.Debug("skip fetching due to accrual pause")
		return
	}

	newOrders, err := f.store.GetNewOrders(ctx)
	if err != nil {
		f.logger.Error("get new orders", zap.Error(err))
		return
	}

	for _, order := range newOrders {
		select {
		case <-ctx.Done():
			f.logger.Info("stop getOrders")
			return
		case f.ordersCh <- order:
		}
	}
}

func (f *ProcessOrder) stuckOrdersWorker(ctx context.Context) {
	const stuckOrderCheckInterval = 1 * time.Minute
	tick := time.NewTicker(stuckOrderCheckInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("stop stuck orders worker")
			return
		case <-tick.C:
			f.resetStuckOrders(ctx)
		}
	}
}

func (f *ProcessOrder) resetStuckOrders(ctx context.Context) {
	const (
		stuckOrderTimeout    = 5 * time.Minute
		stuckOrderBatchLimit = 1000
	)
	err := f.store.ResetStuckOrders(ctx, stuckOrderTimeout, stuckOrderBatchLimit)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			f.logger.Error("failed to reset stuck orders", zap.Error(err))
		}
		return
	}
}

func (f *ProcessOrder) Close() {
	f.closeOnce.Do(func() {
		close(f.ordersCh)
		f.wg.Wait()
		f.logger.Debug("ProcessOrder closed")
	})
}
