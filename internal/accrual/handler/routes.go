package handler

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func addRoutes(
	mux *chi.Mux,
	logger *zap.Logger,
	_ *config.Config,
) {
	mux.Use(middleware.NewGzip(logger))

	mux.Route("/api", func(mux chi.Router) {
		// mux.Post("/goods", handleGoogs)
		// mux.Route("/orders", func(mux chi.Mux) {
		// 	mux.Post("/", handleOrders)
		// 	mux.Get("/{number}", handleOrderNumber)
		// })
	})
}
