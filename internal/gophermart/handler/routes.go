package handler

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/orders"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/ping"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	localmiddleware "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/middleware"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/gophermart"
	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"
)

func addRoutes(
	mux *chi.Mux,
	logger *zap.Logger,
	_ *config.Config,
	gophermart *gophermart.Gophermart,
) {
	mux.Use(localmiddleware.RequestLogger(logger))
	mux.Use(localmiddleware.GzipMiddleware(logger))

	mux.Route("/api", func(mux chi.Router) {
		mux.Get("/ping", ping.Ping(logger, gophermart))

		mux.Route("/user", func(mux chi.Router) {
			mux.Route("/orders", func(mux chi.Router) {
				mux.Post("/", orders.Store(logger, gophermart))
			})
		})
	})
}
