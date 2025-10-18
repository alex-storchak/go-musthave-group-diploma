package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	config "github.com/alex-storchak/go-musthave-group-diploma/internal/config/accrual"
	logger "github.com/alex-storchak/go-musthave-group-diploma/internal/logger/accrual"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	zl, err := initLogger(cfg)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	// defer func() {
	//	if sErr := zl.Sync(); sErr != nil {
	//		fmt.Fprintf(os.Stderr, "logger sync error: %v\n", sErr)
	//	}
	// }()

	ctx := context.Background()
	if err := run(ctx, cfg, zl); err != nil {
		zl.Error("failed to run application", zap.Error(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, zl *zap.Logger) error {
	_, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	zl.Debug("config", zap.Any("config", cfg))

	// log.Printf("starting server on %s, log.level=%s", cfg.Addr(), cfg.Log.Level)
	//
	// srv := &http.Server{
	// 	Addr: cfg.Addr(),
	// }
	// fmt.Println("Listening on", cfg.Addr())
	// if err := srv.ListenAndServe(); err != nil {
	// 	log.Fatal(err)
	// }

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
