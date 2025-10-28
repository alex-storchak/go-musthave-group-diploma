package handler

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance/withdrawals"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/orders"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/ping"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	localmiddleware "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/middleware"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/gophermart"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func addRoutes(
	mux *chi.Mux,
	logger *zap.Logger,
	cfg *config.Config,
	gophermart *gophermart.Gophermart,
) {
	mux.Use(localmiddleware.RequestLogger(logger))
	mux.Use(middleware.Compress(cfg.CompressLevel))

	mux.Route("/api", func(mux chi.Router) {
		mux.Get("/ping", ping.Ping(logger, gophermart))

		mux.Route("/user", func(mux chi.Router) {
			mux.Route("/orders", func(mux chi.Router) {
				mux.Get("/", orders.Index(logger, gophermart))
				mux.Post("/", orders.Store(logger, gophermart))
			})

			mux.Route("/balance", func(mux chi.Router) {
				mux.Get("/", balance.Show(logger, gophermart))
				mux.Post("/withdraw", withdrawals.Store(logger, gophermart))
			})
		})
	})
}
