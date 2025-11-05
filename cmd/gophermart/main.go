package main

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/db"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/logging"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/gophermart"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/worker"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"log"
	"os"
	"os/signal"
)

func run(ctx context.Context, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cfg, err := config.GetConfig(args[1:])
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	zl, err := logging.Initialize(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		//nolint:errcheck // there isn't any good strategy to log error
		_ = zl.Sync()
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

	userRepo := pg.NewPgUser(conn)
	userProvider := service.NewAuthUser(userRepo)
	tokenProvider := service.NewAuthToken(cfg.Handlers)
	auth := service.NewAuth(userProvider, tokenProvider)

	router := handler.NewRouter(zl, cfg.Handlers, gmart, auth)

	processOrder, err := worker.NewProcessOrder(conn, cfg, zl)
	if err != nil {
		return fmt.Errorf("new process order: %w", err)
	}
	go processOrder.StartProcessOrder(ctx)

	handler.Serve(ctx, zl, cfg.Handlers, router)
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args); err != nil {
		log.Fatalf("failed to run application: %v", err)
	}
}
