package main

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/logging"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
	"os"
	"os/signal"
)

func run(ctx context.Context, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cfg, err := config.GetConfig(args)
	if err != nil {
		return err
	}

	logger, err := logging.Initialize(cfg)
	if err != nil {
		return err
	}

	gophermartService, err := service.NewGophermart(cfg.Handlers)
	if err != nil {
		return err
	}

	defer gophermartService.Close()

	return handler.Serve(ctx, logger, cfg.Handlers, gophermartService)
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
