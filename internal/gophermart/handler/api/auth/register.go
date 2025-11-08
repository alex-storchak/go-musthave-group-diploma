package handlerauth

import (
	"context"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	utils "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"go.uber.org/zap"
	"net/http"
)

type UserRegisterer interface {
	Register(ctx context.Context, login, password string) (*models.User, string, error)
}

//nolint:dupl // register and login are different business processes with possible same structure
func HandleRegister(cfg *config.Config, l *zap.Logger, reg UserRegisterer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		creds, err := codec.Decode[models.AuthRequest](r)
		if err != nil {
			l.Debug("decode json request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = validator.IsValid(creds)
		if err != nil {
			l.Debug("check validity", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, token, err := reg.Register(context.Background(), creds.Login, creds.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserAlreadyExists) {
				w.WriteHeader(http.StatusConflict)
				return
			}

			l.Error("register user via service", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		utils.SetAuthCookie(cfg, w, token)

		w.WriteHeader(http.StatusOK)
	}
}
