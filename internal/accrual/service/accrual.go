package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
)

var (
	ErrOrderAlreadyRegistered = errors.New("order already registered")
)

type OrdersRepository interface {
	Add(ctx context.Context, order model.Order) error
	Has(ctx context.Context, number string) (bool, error)
}

type Accrual struct {
	orders OrdersRepository
}

func NewAccrual(orders OrdersRepository) *Accrual {
	return &Accrual{
		orders: orders,
	}
}

func (a *Accrual) RegisterOrder(ctx context.Context, order model.Order) error {
	has, err := a.orders.Has(ctx, order.Number)
	if err != nil {
		return fmt.Errorf("check if order exists in repo: %w", err)
	}
	if has {
		return ErrOrderAlreadyRegistered
	}

	if err := a.orders.Add(ctx, order); err != nil {
		return fmt.Errorf("add order to repo: %w", err)
	}
	return nil
}
