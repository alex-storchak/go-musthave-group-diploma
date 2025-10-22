package handler

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const mediumCompressLevel int = 5

func addRoutes(
	mux *chi.Mux,
	logger *zap.Logger,
	_ *config.Config,
	accrual *service.Accrual,
) {
	mux.Use(middleware.Compress(mediumCompressLevel))

	mux.Route("/api", func(mux chi.Router) {
		mux.Post("/goods", handleRewardRule(logger, accrual))
		mux.Route("/orders", func(mux chi.Router) {
			mux.Post("/", handleOrders(logger, accrual))
			// mux.Get("/{number}", handleOrderNumber)
		})
	})
}
