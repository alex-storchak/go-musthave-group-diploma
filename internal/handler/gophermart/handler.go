package handler

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/handler/gophermart/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/handler/gophermart/middleware"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
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
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(
		mux,
		logger,
		cfg,
		gophermart,
	)
	var handler http.Handler = mux
	handler = middleware.GzipMiddleware(logger)(handler)
	handler = middleware.RequestLogger(logger)(handler)
	return handler
}

func addRoutes(
	mux *http.ServeMux,
	logger *zap.Logger,
	cfg *config.Config,
	gophermart Gophermart,
) {
	mux.Handle("/api/v1/", handleTenantsGet(logger, tenantsStore))
	mux.Handle("/oauth2/", handleOAuth2Proxy(logger, authProxy))
	mux.HandleFunc("/healthz", handleHealthzPlease(logger))
	mux.Handle("/", http.NotFoundHandler())
}

func Serve(ctx context.Context, logger *zap.Logger, cfg *config.Config, gophermart Gophermart) error {
	srv := newHandlers(
		logger,
		cfg,
		gophermart,
	)
	httpServer := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: srv,
	}
	go func() {
		logger.Info("starting server", zap.String("addr", cfg.ServerAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	}()
	wg.Wait()
	return nil
}

func Ping(w http.ResponseWriter, r *http.Request) {
	err := h.shortener.Ping(r.Context())
	if err != nil {
		logger.Log.Error("ping store error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
