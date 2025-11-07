package worker

import (
	"context"
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
	sleepSearchOrders       = 50 * time.Millisecond
	ordersBufferSize        = 50
	getOrdersTickerDuration = 500 * time.Millisecond
)

type Accrual interface {
	Get(ctx context.Context, order models.OrderProcess) (*models.AccrualResponse, error)
	GetRetryAfter() time.Duration
	SetRetryAfter(t time.Duration)
}

type ProcessOrder struct {
	store         repository.Repository
	accrual       Accrual
	logger        *zap.Logger
	ordersCh      chan models.OrderProcess
	processingIDs map[string]struct{}
	mu            *sync.RWMutex
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
		store:         store,
		accrual:       acc,
		logger:        l,
		ordersCh:      make(chan models.OrderProcess, ordersBufferSize),
		processingIDs: make(map[string]struct{}),
		mu:            &sync.RWMutex{},
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
	done := make(chan struct{}, maxConcurrent)
	errCh := make(chan error, maxConcurrent)

	go f.RunGetOrders(ctx)

	var pauseTimer *time.Timer

	go func() {
		for {
			select {
			case <-ctx.Done():
				f.logger.Info("StartProcessOrder: context canceled")
				return

			case err := <-errCh:
				f.logger.Error("error accrual processing", zap.Error(err))
				if f.accrual.GetRetryAfter() > 0 {
					// Останавливаем текущий таймер, если есть
					if pauseTimer != nil {
						pauseTimer.Stop()
					}
					// Запускаем новый таймер
					pauseTimer = time.AfterFunc(f.accrual.GetRetryAfter(), func() {
						f.accrual.SetRetryAfter(0)
						f.logger.Info("pause processing accrual ended")
					})
				}

			case <-done:
				f.doneProcessing(ctx, done, errCh)
			}
		}
	}()

	for i := 0; i < maxConcurrent; i++ {
		done <- struct{}{}
	}
}

func (f *ProcessOrder) doneProcessing(
	ctx context.Context,
	done chan<- struct{},
	errCh chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			f.logger.Info("stop search order")
			return

		case order := <-f.ordersCh:
			go f.startAccrualWorker(ctx, &order, done, errCh)
			return
		}
	}
}

func (f *ProcessOrder) startAccrualWorker(
	ctx context.Context,
	order *models.OrderProcess,
	done chan<- struct{},
	errCh chan<- error,
) {
	defer func() {
		done <- struct{}{}
	}()

	// 1. Получение данных начисления
	accrualResponse, err := f.accrual.Get(ctx, *order)
	if err != nil {
		errCh <- fmt.Errorf("accrual get failed: %w", err)
		return
	}

	if accrualResponse == nil {
		f.logger.Warn("accrual response is nil", zap.String("order_number", order.Number))
		return
	}

	// 2. Обработка статусов через switch (нагляднее if-else)
	switch accrualResponse.Status {
	case models.AccrualRegistered, models.AccrualProcessing, models.AccrualInvalid:
		// Для этих статусов логика одинаковая
		if accrualResponse.Status == models.AccrualInvalid {
			if err := f.store.UpdateOrderInvalid(ctx, accrualResponse); err != nil {
				errCh <- fmt.Errorf("update invalid order: %w", err)
			}
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

	f.mu.Lock()
	delete(f.processingIDs, order.Number)
	f.mu.Unlock()
}

func (f *ProcessOrder) RunGetOrders(ctx context.Context) {
	ticker := time.NewTicker(getOrdersTickerDuration)
	defer ticker.Stop()

	f.logger.Info("start process order")

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
	f.mu.Lock()
	defer f.mu.Unlock()

	currentIDs := make([]string, 0, len(f.processingIDs))
	for id := range f.processingIDs {
		currentIDs = append(currentIDs, id)
	}

	newOrders, err := f.store.GetNewOrders(ctx, currentIDs)
	if err != nil {
		f.logger.Error("get new orders", zap.Error(err))
		return
	}

	for _, order := range newOrders {
		select {
		case <-ctx.Done():
			return
		case f.ordersCh <- order:
			f.processingIDs[order.Number] = struct{}{}
		default:
			// Канал переполнен — пропускаем (или логируем)
			f.logger.Warn("orders channel full, skipping order", zap.String("number", order.Number))
		}
	}
}
