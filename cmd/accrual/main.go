package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/logger"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository/factory"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/worker"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalf("failed to run application: %v", err)
	}
}

func run(
	ctx context.Context,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	zl, err := initLogger(cfg)
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer func() {
		_ = zl.Sync()
	}()

	pgFactory, err := factory.NewPgRepo(ctx, cfg.DB, zl)
	if err != nil {
		return fmt.Errorf("create pg repo factory: %w", err)
	}

	orders := pgFactory.MakeOrders()
	rules := pgFactory.MakeRewardRules()

	rulesProvider := service.NewCacheRulesProvider(rules, cfg.Accrual.RulesCacheTTL, zl)
	defer rulesProvider.Close()
	rulesProvider.Start(ctx)

	accrual := service.NewAccrual(orders, rules, zl)
	defer accrual.Close()

	accrualPool := worker.NewAccrualPool(accrual, orders, rulesProvider, &cfg.Accrual, zl)
	defer accrualPool.Close()
	accrualPool.Start(ctx)

	router := handler.NewRouter(zl, cfg, accrual, rulesProvider)
	handler.Serve(ctx, cfg.Server, zl, router)
	return nil
}

func initLogger(cfg *config.Config) (*zap.Logger, error) {
	zl, err := logger.New(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("new logger. config: %v error: %w", cfg.Log, err)
	}
	zl.Info("logger initialized")
	return zl, nil
}
