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
	"time"
)

const (
	maxConcurrent           = 10
	ordersBufferSize        = 50
	getOrdersTickerDuration = 5 * time.Second
)

type Accrual interface {
	Get(ctx context.Context, order models.OrderProcess) (*models.AccrualResponse, error)
	GetRetryAfter() time.Duration
}

type ProcessOrder struct {
	store    repository.Repository
	accrual  Accrual
	logger   *zap.Logger
	ordersCh chan models.OrderProcess
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
	errCh := make(chan error, 2*maxConcurrent)

	go f.RunGetOrders(ctx)

	f.startWorkers(ctx, errCh)

	go f.stuckOrdersWorker(ctx)
}

func (f *ProcessOrder) startWorkers(ctx context.Context, errCh chan<- error) {
	for i := 0; i < maxConcurrent; i++ {
		workerID := i
		go f.worker(ctx, errCh, workerID)
	}
}

func (f *ProcessOrder) worker(ctx context.Context, errCh chan<- error, workerID int) {
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
			case order, ok := <-f.ordersCh:
				if !ok {
					return
				}
				f.processOrder(ctx, order, errCh)
			case <-ctx.Done():
				return
			}
		}
	}
}

func (f *ProcessOrder) waitRetry(ctx context.Context, workerID int) bool {
	d := f.accrual.GetRetryAfter()
	if d <= 0 {
		return true
	}

	f.logger.Debug("worker paused due to retry-after",
		zap.Duration("retry_after", d),
		zap.Int("worker_id", workerID))

	timer := time.NewTimer(d)
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

func (f *ProcessOrder) processOrder(ctx context.Context, order models.OrderProcess, errCh chan<- error) {
	// Получение данных начисления
	accrualResponse, err := f.accrual.Get(ctx, order)
	if err != nil {
		errCh <- fmt.Errorf("accrual get failed: %w", err)
		return
	}

	// Обработка статусов
	switch accrualResponse.Status {
	case models.AccrualRegistered, models.AccrualProcessing:
	case models.AccrualInvalid:
		if err = f.store.UpdateOrderInvalid(ctx, accrualResponse); err != nil {
			errCh <- fmt.Errorf("update invalid order: %w", err)
		}
	case models.AccrualProcessed:
		if err := f.store.UpdateOrderProcessed(ctx, accrualResponse); err != nil {
			f.logger.Error("failed to update processed order",
				zap.String("order_number", accrualResponse.Number),
				zap.Error(err))
			errCh <- fmt.Errorf("update processed order: %w", err)
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
			close(f.ordersCh)
			return
		case <-ticker.C:
			f.GetOrders(ctx)
		}
	}
}

func (f *ProcessOrder) GetOrders(ctx context.Context) {
	if d := f.accrual.GetRetryAfter(); d > 0 {
		f.logger.Debug("skip fetching due to accrual pause", zap.Duration("retry_after", d))
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
