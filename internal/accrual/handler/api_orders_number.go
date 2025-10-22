package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type OrderInformer interface {
	InformOrder(ctx context.Context, number string) (*model.Order, error)
}

func handleOrderNumber(logger *zap.Logger, inf OrderInformer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderNum := chi.URLParam(r, OrderNumberParam)

		order, err := inf.InformOrder(context.Background(), orderNum)
		if err != nil {
			if errors.Is(err, repository.ErrOrderNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Error("get order accrual with accrual service", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = codec.Encode(w, http.StatusOK, order)
		if err != nil {
			logger.Error("encode json response", zap.Error(err))
			return
		}
	}
}
