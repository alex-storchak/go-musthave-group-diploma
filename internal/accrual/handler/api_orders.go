package handler

import (
	"context"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	utils "github.com/alex-storchak/go-musthave-group-diploma/internal/utils/strings"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

const (
	MsgEmptyOrderNumber     = "order is required"
	MsgInvalidOrderNumber   = "order is invalid (Luhn algorithm check failed)"
	MsgEmptyGoodDescription = "description is required"
	MsgInvalidGoodPrice     = "price must be greater than 0"
)

type reqGood struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type reqOrder struct {
	Number string    `json:"order"`
	Goods  []reqGood `json:"goods"`
}

func (o reqOrder) Valid() validator.Problems {
	problems := make(validator.Problems)

	if o.Number == "" {
		problems["order"] = MsgEmptyOrderNumber
	} else if !validator.IsValidLuhn(o.Number) {
		problems["order"] = MsgInvalidOrderNumber
	}

	for i, g := range o.Goods {
		if g.Description == "" {
			problems["goods."+strconv.Itoa(i)+".description"] = MsgEmptyGoodDescription
		}
		if g.Price <= 0 {
			problems["goods."+strconv.Itoa(i)+".price"] = MsgInvalidGoodPrice
		}
	}

	return problems
}

type OrderRegisterer interface {
	RegisterOrder(ctx context.Context, order *model.Order) error
}

func prepareOrder(order reqOrder) *model.Order {
	mo := model.Order{
		Number: utils.RemoveWhitespaces(order.Number),
		Goods:  make([]model.Good, 0, len(order.Goods)),
	}
	for _, g := range order.Goods {
		mg := model.Good{
			Description: g.Description,
			Price:       g.Price,
		}
		mo.Goods = append(mo.Goods, mg)
	}
	return &mo
}

func handleOrders(l *zap.Logger, reg OrderRegisterer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		order, err := codec.Decode[reqOrder](r)
		if err != nil {
			l.Debug("decode json request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = validator.IsValid(order)
		if err != nil {
			l.Debug("check validity", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := reg.RegisterOrder(context.Background(), prepareOrder(order)); err != nil {
			if errors.Is(err, service.ErrOrderAlreadyRegistered) {
				w.WriteHeader(http.StatusConflict)
				return
			}

			l.Error("register order via service", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
