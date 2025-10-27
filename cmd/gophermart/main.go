package main

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/db"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/logging"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/gophermart"
	"gorm.io/gorm"
	"io"
	"os"
	"os/signal"
)

func run(ctx context.Context, stderr io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cfg, err := config.GetConfig(args)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	zl, err := logging.Initialize(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	defer func() {
		if sErr := zl.Sync(); sErr != nil {
			_, _ = fmt.Fprintf(stderr, "logger sync error: %v", sErr)
		}
	}()

	conn, err := db.InitGORMDB(cfg)
	if err != nil {
		return fmt.Errorf("init db error: %w", err)
	}
	defer func(conn *gorm.DB) {
		sqlDB, err := conn.DB()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error getting underlying DB: %v", err)
			return
		}

		err = sqlDB.Close()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "close db error: %v", err)
		}
	}(conn)

	gophermart, err := gophermart.NewGophermart(conn)
	if err != nil {
		return fmt.Errorf("failed to initialize service gophermart: %w", err)
	}

	router := handler.NewRouter(zl, cfg.Handlers, gophermart)

	return handler.Serve(ctx, zl, cfg.Handlers, router)
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
