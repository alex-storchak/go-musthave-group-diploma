package handler

import (
	handlerauth "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance/withdrawals"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/orders"
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
	g *gophermart.Gophermart,
	auth *service.Auth,
) {
	mux.Use(middleware.Logger)
	mux.Use(middleware.Compress(cfg.CompressLevel))

	mux.Route("/api", func(mux chi.Router) {
		mux.Route("/user", func(mux chi.Router) {
			mux.Post("/register", handlerauth.HandleRegister(cfg, logger, auth))
			mux.Post("/login", handlerauth.HandleLogin(cfg, logger, auth))

			// auth protected group
			mux.Group(func(mux chi.Router) {
				mux.Use(mw.NewAuth(cfg, logger, auth))

				mux.Route("/orders", func(mux chi.Router) {
					mux.Get("/", orders.Index(logger, g))
					mux.Post("/", orders.Store(logger, g))
				})

				mux.Route("/balance", func(mux chi.Router) {
					mux.Get("/", balance.Show(logger, g))
					mux.Post("/withdraw", withdrawals.Store(logger, g))
				})

				mux.Get("/withdrawals", withdrawals.Index(logger, g))
			})
		})
	})
}
