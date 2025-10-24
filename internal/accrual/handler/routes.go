package handler

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/worker"
	mw "github.com/alex-storchak/go-musthave-group-diploma/internal/middleware"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	OrderNumberParam = "number"

	mediumCompressLevel int = 5
)

func addRoutes(
	mux *chi.Mux,
	logger *zap.Logger,
	cfg *config.Config,
	accrual *service.Accrual,
	accrualPool *worker.AccrualPool,
) {
	mux.Use(middleware.Logger)
	mux.Use(middleware.Compress(mediumCompressLevel))

	rateLimiter := mw.NewEndpointRateLimiter(cfg.Server.RequestsRateLimit, logger)
	mux.Route("/api", func(mux chi.Router) {
		mux.Post("/goods", handleRewardRule(logger, accrual, accrualPool))
		mux.Route("/orders", func(mux chi.Router) {
			mux.Post("/", handleOrders(logger, accrual))
			mux.With(rateLimiter).Get("/{number}", handleOrderNumber(logger, accrual))
		})
	})
}
