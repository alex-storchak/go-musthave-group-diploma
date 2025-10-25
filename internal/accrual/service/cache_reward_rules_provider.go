package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"go.uber.org/zap"
)

type CacheRulesProvider struct {
	repo       RulesRepository
	cache      atomic.Pointer[[]model.RewardRule]
	cacheDirty atomic.Bool
	cacheTTL   time.Duration
	cacheMu    *sync.Mutex
	logger     *zap.Logger
	wg         sync.WaitGroup
	started    atomic.Bool
	closed     atomic.Bool
}

func NewCacheRulesProvider(
	repo RulesRepository,
	cacheTTL time.Duration,
	l *zap.Logger,
) *CacheRulesProvider {
	rp := &CacheRulesProvider{
		repo:     repo,
		cacheTTL: cacheTTL,
		cacheMu:  &sync.Mutex{},
		logger:   l,
	}
	rp.clearCache()

	return rp
}

func (p *CacheRulesProvider) Start(ctx context.Context) {
	if !p.started.CompareAndSwap(false, true) {
		return
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.startCacheRefresher(ctx)
	}()

	p.logger.Info("rules cache refresher started", zap.Any("cache_ttl", p.cacheTTL))
}

func (p *CacheRulesProvider) startCacheRefresher(ctx context.Context) {
	p.refreshCache(ctx)

	ticker := time.NewTicker(p.cacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("rules cache refresher shutting down")
			return
		case <-ticker.C:
			p.refreshCache(ctx)
		}
	}
}

func (p *CacheRulesProvider) refreshCache(ctx context.Context) {
	rules, err := p.ensureFreshRules(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			p.logger.Error("failed to refresh rules cache", zap.Error(err))
		}
		return
	}
	p.logger.Debug("rules cache updated by ticker", zap.Int("rules_count", len(rules)))
}

func (p *CacheRulesProvider) MarkDirty() {
	p.cacheDirty.Store(true)
	p.logger.Debug("rules cache marked as dirty")
}

func (p *CacheRulesProvider) clearCache() {
	p.cacheDirty.Store(false)
	emptyRules := make([]model.RewardRule, 0)
	p.cache.Store(&emptyRules)
	p.logger.Debug("rules cache cleared")
}

func (p *CacheRulesProvider) updateCache(rules []model.RewardRule) {
	newRules := make([]model.RewardRule, len(rules))
	copy(newRules, rules)
	p.cache.Store(&newRules)
}

func (p *CacheRulesProvider) All(ctx context.Context) ([]model.RewardRule, error) {
	// Первая критическая секция: быстрый чек кэша
	p.cacheMu.Lock()
	dirty := p.cacheDirty.Load()
	cachedPtr := p.cache.Load()
	if !dirty && cachedPtr != nil && len(*cachedPtr) > 0 {
		rules := *cachedPtr
		p.cacheMu.Unlock()
		return rules, nil
	}
	p.cacheMu.Unlock()

	rulesFromRepo, err := p.ensureFreshRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all rules from repo: %w", err)
	}
	return rulesFromRepo, nil

}

// Делает I/O и безопасно обновляет кэш с double-check, возвращает актуальные правила.
func (p *CacheRulesProvider) ensureFreshRules(ctx context.Context) ([]model.RewardRule, error) {
	rulesFromRepo, err := p.repo.All(ctx)
	if err != nil {
		return nil, err
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	dirty := p.cacheDirty.Load()
	cachedPtr := p.cache.Load()
	if !dirty && cachedPtr != nil && len(*cachedPtr) > 0 {
		return *cachedPtr, nil
	}
	p.updateCache(rulesFromRepo)
	p.cacheDirty.Store(false)
	p.logger.Debug("rules cache updated")
	return rulesFromRepo, nil
}

func (p *CacheRulesProvider) Close() {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	p.wg.Wait()
	p.logger.Info("cache rules provider closed")
}
