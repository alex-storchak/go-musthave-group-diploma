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
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	zl, err := initLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		if sErr := zl.Sync(); sErr != nil {
			_, _ = fmt.Fprintf(stderr, "logger sync error: %v\n", sErr)
		}
	}()

	router := handler.NewRouter(zl, cfg)
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
