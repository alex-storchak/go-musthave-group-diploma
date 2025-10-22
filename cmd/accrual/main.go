package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/logger"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository/factory"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stderr); err != nil {
		log.Fatalf("failed to run application: %v", err)
	}
}

func run(
	ctx context.Context,
	stderr io.Writer,
) error {
	_, cancel := signal.NotifyContext(ctx, os.Interrupt)
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
		if sErr := zl.Sync(); sErr != nil {
			_, _ = fmt.Fprintf(stderr, "logger sync error: %v\n", sErr)
		}
	}()

	pgFactory, err := factory.NewPgRepo(ctx, cfg.DB, zl)
	if err != nil {
		return fmt.Errorf("create pg repo factory: %w", err)
	}
	accrual := initAccrual(pgFactory, zl)
	defer accrual.Close()

	router := handler.NewRouter(zl, cfg, accrual)
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

func initAccrual(f factory.Repo, l *zap.Logger) *service.Accrual {
	o := f.MakeOrders()
	r := f.MakeRewardRules()
	return service.NewAccrual(o, r, l)
}
