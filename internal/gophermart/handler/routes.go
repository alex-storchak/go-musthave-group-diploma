package handler

import (
	handlerauth "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance/withdrawals"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/orders"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/ping"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	mw "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/middleware"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
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
	auth *service.Auth,
) {
	mux.Use(middleware.Logger)
	mux.Use(middleware.Compress(cfg.CompressLevel))

	mux.Route("/api", func(mux chi.Router) {
		mux.Get("/ping", ping.Ping(logger, gophermart))

		mux.Route("/user", func(mux chi.Router) {
			mux.Post("/register", handlerauth.HandleRegister(cfg, logger, auth))
			mux.Post("/login", handlerauth.HandleLogin(cfg, logger, auth))

			// auth protected group
			mux.Group(func(mux chi.Router) {
				mux.Use(mw.NewAuth(cfg, logger, auth))

				mux.Route("/orders", func(mux chi.Router) {
					mux.Get("/", orders.Index(logger, gophermart))
					mux.Post("/", orders.Store(logger, gophermart))
				})

				mux.Route("/balance", func(mux chi.Router) {
					mux.Get("/", balance.Show(logger, gophermart))
					mux.Post("/withdraw", withdrawals.Store(logger, gophermart))
				})

				mux.Get("/withdrawals", withdrawals.Index(logger, gophermart))
			})
		})
	})
}
