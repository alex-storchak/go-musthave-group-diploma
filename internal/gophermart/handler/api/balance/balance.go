package balance

import (
	"context"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	_ "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Gophermart interface {
	GetBalance(ctx context.Context, getBalance models.GetBalanceRequest) (*models.ShowBalanceResponse, error)
	CreateDefaultBalance(ctx context.Context, setDefaultBalance models.SetDefaultBalanceRequest) error
}

func Show(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := int64(1)
		indexOrder := models.GetBalanceRequest{UserID: userID}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		balance, err := gophermart.GetBalance(ctx, indexOrder)
		if err != nil {
			if errors.Is(err, myerrors.ErrBalanceNotFound) {
				err = gophermart.CreateDefaultBalance(ctx, models.SetDefaultBalanceRequest{
					UserID:         userID,
					Current:        0,
					TotalWithdrawn: 0,
				})
				if err != nil {
					logger.Error("create default balance", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				balance = &models.ShowBalanceResponse{
					Current:        0,
					TotalWithdrawn: 0,
				}
			} else {
				logger.Error("get balance", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		err = codec.Encode(w, http.StatusOK, balance)
		if err != nil {
			logger.Error("encode json response", zap.Error(err))
			return
		}
	}
}
