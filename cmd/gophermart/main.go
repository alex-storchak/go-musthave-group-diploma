package main

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/logging"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
	"io"
	"os"
	"os/signal"
)

func run(ctx context.Context, w io.Writer, args []string) error {
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

	defer func(gophermartService *service.Gophermart) {
		err := gophermartService.Close()
		if err != nil {

		}
	}(gophermartService)

	return handler.Serve(ctx, logger, cfg.Handlers, gophermartService)
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
