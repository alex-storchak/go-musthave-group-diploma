package handler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	localmiddleware "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Gophermart interface {
	Close() error
	Ping(ctx context.Context) error
}

type handlers struct {
	logger     *zap.Logger
	cfg        *config.Config
	gophermart Gophermart
}

func newHandlers(
	logger *zap.Logger,
	cfg *config.Config,
	gophermart Gophermart,
) *handlers {
	return &handlers{
		logger:     logger,
		cfg:        cfg,
		gophermart: gophermart,
	}
}

func newRouter(h *handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Compress(5))
	r.Use(localmiddleware.RequestLogger(h.logger))

	r.Get("/ping", h.Ping)

	return r
}

func Serve(ctx context.Context, logger *zap.Logger, cfg *config.Config, gophermart Gophermart) error {
	h := newHandlers(logger, cfg, gophermart)
	router := newRouter(h)

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

func (h *handlers) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.gophermart.Ping(r.Context())
	if err != nil {
		h.logger.Error("ping store error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
