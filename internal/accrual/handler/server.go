package handler

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func NewRouter(
	logger *zap.Logger,
	cfg *config.Config,
	accrual *service.Accrual,
) http.Handler {
	r := chi.NewRouter()
	addRoutes(r, logger, cfg, accrual)
	return r
}

func Serve(
	ctx context.Context,
	cfg config.Server,
	logger *zap.Logger,
	router http.Handler,
) {
	httpServer := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}
	go func() {
		logger.Info("starting server", zap.String("run address", cfg.RunAddress))
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
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, cfg.ShutdownWaitSecsDuration)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down http server", zap.Error(err))
		}
	}()
	wg.Wait()
}
