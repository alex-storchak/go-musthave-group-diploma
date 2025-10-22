package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"go.uber.org/zap"
)

var ErrOrderAlreadyRegistered = errors.New("order already registered")

type OrdersRepository interface {
	Add(ctx context.Context, order *model.Order) error
	Has(ctx context.Context, number string) (bool, error)
	Close()
}

type Accrual struct {
	Orders OrdersRepository
	logger *zap.Logger
}

func NewAccrual(orders OrdersRepository, l *zap.Logger) *Accrual {
	return &Accrual{
		Orders: orders,
		logger: l,
	}
}

func (a *Accrual) RegisterOrder(ctx context.Context, order *model.Order) error {
	has, err := a.Orders.Has(ctx, order.Number)
	if err != nil {
		return fmt.Errorf("check if order exists in repo: %w", err)
	}
	if has {
		return ErrOrderAlreadyRegistered
	}

	if err := a.Orders.Add(ctx, order); err != nil {
		return fmt.Errorf("add order to repo: %w", err)
	}
	return nil
}
