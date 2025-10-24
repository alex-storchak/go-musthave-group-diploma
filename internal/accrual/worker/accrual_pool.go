package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"go.uber.org/zap"
)

type OrderProcessor interface {
	ProcessOrder(order *model.Order, rules []model.RewardRule)
}

type ProcessOrdersRepository interface {
	GetBatchForProcessing(ctx context.Context, batchSize int) ([]model.Order, error)
	UpdateStatus(ctx context.Context, order *model.Order) error
	ResetStuckOrders(ctx context.Context, timeout time.Duration, batchLimit int) (int, error)
}

type AccrualPool struct {
	processor       OrderProcessor
	orders          ProcessOrdersRepository
	rules           service.RulesRepository
	rulesCache      atomic.Pointer[[]model.RewardRule]
	rulesCacheDirty atomic.Bool
	rulesCacheMu    *sync.Mutex
	cfg             config.Accrual
	jobChan         chan *model.Order
	logger          *zap.Logger
	wg              sync.WaitGroup
	started         atomic.Bool
	closed          atomic.Bool
}

func NewAccrualPool(
	p OrderProcessor,
	o ProcessOrdersRepository,
	r service.RulesRepository,
	cfg *config.Accrual,
	l *zap.Logger,
) *AccrualPool {
	ap := &AccrualPool{
		processor:    p,
		orders:       o,
		rules:        r,
		rulesCacheMu: &sync.Mutex{},
		jobChan:      make(chan *model.Order, cfg.JobChanSize),
		cfg:          *cfg,
		logger:       l,
	}

	ap.ClearRulesCache()

	return ap
}

func (p *AccrualPool) Start(ctx context.Context) {
	if !p.started.CompareAndSwap(false, true) {
		return
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.startCacheRefresher(ctx)
	}()

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
}

func (p *AccrualPool) dispatcher(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("accrual worker pool shutting down")
			close(p.jobChan)
			return
		case <-ticker.C:
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
		if ctx.Err() != nil {
			p.logger.Debug("ctx done, stop sending orders to jobChannel")
			return
		}

		select {
		case p.jobChan <- &orders[i]:
		case <-ctx.Done():
			p.logger.Debug("ctx done while sending, stop sending orders")
			return
		}
	}
}

func (p *AccrualPool) worker(ctx context.Context, workerID int) {
	for order := range p.jobChan {
		rules, err := p.getAllRules(ctx)
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
	ticker := time.NewTicker(p.cfg.StuckOrderCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("stuck orders worker shutting down")
			return
		case <-ticker.C:
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

func (p *AccrualPool) getAllRules(ctx context.Context) ([]model.RewardRule, error) {
	p.rulesCacheMu.Lock()
	defer p.rulesCacheMu.Unlock()

	if p.rulesCacheDirty.Load() {
		p.ClearRulesCache()
	}

	cached := p.rulesCache.Load()
	if cached != nil && len(*cached) > 0 {
		return *cached, nil
	}

	rules, err := p.rules.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all rules from repo: %w", err)
	}

	p.updateCache(rules)
	return rules, nil
}

func (p *AccrualPool) updateCache(rules []model.RewardRule) {
	newRules := make([]model.RewardRule, len(rules))
	copy(newRules, rules)
	p.rulesCache.Store(&newRules)
}

func (p *AccrualPool) startCacheRefresher(ctx context.Context) {
	p.refreshCache(ctx)

	ticker := time.NewTicker(p.cfg.RulesCacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.refreshCache(ctx)
		}
	}
}

func (p *AccrualPool) refreshCache(ctx context.Context) {
	rules, err := p.rules.All(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			p.logger.Error("failed to refresh rules cache", zap.Error(err))
		}
		return
	}

	p.updateCache(rules)
	p.logger.Debug("rules cache updated", zap.Int("rules_count", len(rules)))
}

func (p *AccrualPool) MarkCacheDirty() {
	p.rulesCacheDirty.Store(true)
	p.logger.Debug("rules cache marked as dirty")
}

func (p *AccrualPool) ClearRulesCache() {
	p.rulesCacheMu.Lock()
	defer p.rulesCacheMu.Unlock()

	p.rulesCacheDirty.Store(false)
	emptyRules := make([]model.RewardRule, 0)
	p.rulesCache.Store(&emptyRules)
	p.logger.Debug("rules cache cleared")
}

func (p *AccrualPool) Close() {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	p.wg.Wait()
}
