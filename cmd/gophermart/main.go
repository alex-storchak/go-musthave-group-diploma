package main

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/db"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/logging"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/gophermart"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	"log"
	"os"
	"os/signal"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stderr, os.Args); err != nil {
		log.Fatalf("failed to run application: %v", err)
	}
}

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
		sqlDB, cErr := conn.DB()
		if cErr != nil {
			zl.Error("error getting underlying DB", zap.Error(cErr))
			return
		}

		cErr = sqlDB.Close()
		if cErr != nil {
			zl.Error("close db Error", zap.Error(cErr))
		}
	}(conn)

	gmart, err := gophermart.NewGophermart(conn)
	if err != nil {
		return fmt.Errorf("failed to initialize service gophermart: %w", err)
	}

	auth := service.NewAuth(cfg.Handlers, conn)

	router := handler.NewRouter(zl, cfg.Handlers, gmart, auth)

	return handler.Serve(ctx, zl, cfg.Handlers, router)
}
