package handler

import (
	"context"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/gophermart"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

func NewRouter(
	logger *zap.Logger,
	cfg *config.Config,
	gophermart *gophermart.Gophermart,
) http.Handler {
	r := chi.NewRouter()
	addRoutes(r, logger, cfg, gophermart)
	return r
}

func Serve(
	ctx context.Context,
	logger *zap.Logger,
	cfg *config.Config,
	router http.Handler,
) error {
	httpServer := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}
	go func() {
		logger.Info("starting server", zap.String("addr", cfg.ServerAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("error starting server", zap.Error(err))
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down http server", zap.Error(err))
		}
		logger.Info("close server", zap.String("addr", cfg.ServerAddr))
	}()
	wg.Wait()
	return nil
}
