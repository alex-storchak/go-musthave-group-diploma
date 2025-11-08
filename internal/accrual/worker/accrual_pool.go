package worker

import (
	"context"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/ticker"
	"go.uber.org/zap"
	"sync"
	"time"
)

type OrderProcessor interface {
	ProcessOrder(order *model.Order, rules []model.RewardRule)
}

type ProcessOrdersRepository interface {
	GetBatchForProcessing(ctx context.Context, batchSize int) ([]model.Order, error)
	UpdateStatus(ctx context.Context, order *model.Order) error
	ResetStuckOrders(ctx context.Context, timeout time.Duration, batchLimit int) (int, error)
}

type RulesProvider interface {
	All(ctx context.Context) ([]model.RewardRule, error)
	MarkDirty()
	Close()
}

type AccrualPool struct {
	processor     OrderProcessor
	orders        ProcessOrdersRepository
	rules         RulesProvider
	tickerFactory ticker.Factory
	cfg           config.Accrual
	jobChan       chan *model.Order
	logger        *zap.Logger
	wg            sync.WaitGroup
	startOnce     sync.Once
	closeOnce     sync.Once
}

func NewAccrualPool(
	p OrderProcessor,
	o ProcessOrdersRepository,
	r RulesProvider,
	cfg *config.Accrual,
	l *zap.Logger,
	tf ticker.Factory,
) *AccrualPool {
	if tf == nil {
		tf = ticker.RealTickerFactory{}
	}
	return &AccrualPool{
		processor:     p,
		orders:        o,
		rules:         r,
		tickerFactory: tf,
		jobChan:       make(chan *model.Order, cfg.JobChanSize),
		cfg:           *cfg,
		logger:        l,
	}
}

func (p *AccrualPool) Start(ctx context.Context) {
	p.startOnce.Do(func() {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.dispatcher(ctx)
		}()

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.stuckOrdersWorker(ctx)
		}()

		p.wg.Add(p.cfg.WorkerCount)
		for i := 0; i < p.cfg.WorkerCount; i++ {
			go func(i int) {
				defer p.wg.Done()
				p.worker(ctx, i)
			}(i)
		}

		p.logger.Info("accrual pool started", zap.Any("config", p.cfg))
	})
}

func (p *AccrualPool) dispatcher(ctx context.Context) {
	tick := time.NewTicker(p.cfg.PollInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("accrual worker pool shutting down")
			close(p.jobChan)
			return
		case <-tick.C:
			p.dispatchBatch(ctx)
		}
	}
}

func (p *AccrualPool) dispatchBatch(ctx context.Context) {
	orders, err := p.orders.GetBatchForProcessing(ctx, p.cfg.BatchSize)
	if err != nil {
		p.logger.Error("accrual worker pool failed get batch for processing", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		p.logger.Debug("no orders to process, continue")
		return
	}

	p.logger.Debug("processing orders", zap.Int("orders_count", len(orders)))

	for i := range orders {
		select {
		case <-ctx.Done():
			p.logger.Debug("ctx done while sending, stop sending orders")
			return
		case p.jobChan <- &orders[i]:
		}
	}
}

func (p *AccrualPool) worker(ctx context.Context, workerID int) {
	for order := range p.jobChan {
		rules, err := p.rules.All(ctx)
		if err != nil {
			p.logger.Error("accrual worker pool failed to get all rules",
				zap.Int("worker_id", workerID),
				zap.String("order_number", order.Number),
				zap.Error(err),
			)
			p.updateOrderStatus(ctx, order, workerID)
			continue
		}
		p.processor.ProcessOrder(order, rules)
		p.updateOrderStatus(ctx, order, workerID)
	}
}

func (p *AccrualPool) updateOrderStatus(ctx context.Context, order *model.Order, workerID int) {
	if err := p.orders.UpdateStatus(ctx, order); err != nil {
		p.logger.Error("error on update order status",
			zap.Int("worker_id", workerID),
			zap.String("order_number", order.Number),
			zap.String("status", order.Status),
			zap.Error(err),
		)
	}
	p.logger.Debug("worker finished to process order",
		zap.Int("worker_id", workerID),
		zap.String("order_number", order.Number),
	)
}

func (p *AccrualPool) stuckOrdersWorker(ctx context.Context) {
	tick := p.tickerFactory.New(p.cfg.StuckOrderCheckInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("stuck orders worker shutting down")
			return
		case <-tick.C():
			p.resetStuckOrders(ctx)
		}
	}
}

func (p *AccrualPool) resetStuckOrders(ctx context.Context) {
	resetCount, err := p.orders.ResetStuckOrders(ctx, p.cfg.StuckOrderTimeout, p.cfg.StuckOrderBatchLimit)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			p.logger.Error("failed to reset stuck orders", zap.Error(err))
		}
		return
	}

	if resetCount > 0 {
		p.logger.Info("reset stuck orders", zap.Int("reset_count", resetCount))
	}
}

func (p *AccrualPool) Close() {
	p.closeOnce.Do(func() {
		p.wg.Wait()
		p.rules.Close()
		p.logger.Info("accrual worker pool closed")
	})
}
