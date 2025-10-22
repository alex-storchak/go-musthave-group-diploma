package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"go.uber.org/zap"
)

var (
	ErrOrderAlreadyRegistered = errors.New("order already registered")
	ErrRuleAlreadyRegistered  = errors.New("rule already registered")
)

type OrdersRepository interface {
	Add(ctx context.Context, order *model.Order) error
	Has(ctx context.Context, number string) (bool, error)
	Close()
}

type RulesRepository interface {
	Add(ctx context.Context, rule *model.RewardRule) error
	Has(ctx context.Context, match string) (bool, error)
	Close()
}

type Accrual struct {
	Orders OrdersRepository
	Rules  RulesRepository
	logger *zap.Logger
}

func NewAccrual(o OrdersRepository, r RulesRepository, l *zap.Logger) *Accrual {
	return &Accrual{
		Orders: o,
		Rules:  r,
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

func (a *Accrual) RegisterRule(ctx context.Context, rule *model.RewardRule) error {
	has, err := a.Rules.Has(ctx, rule.Match)
	if err != nil {
		return fmt.Errorf("check if rule exists in repo: %w", err)
	}
	if has {
		return ErrRuleAlreadyRegistered
	}

	if err := a.Rules.Add(ctx, rule); err != nil {
		return fmt.Errorf("add rule to repo: %w", err)
	}
	return nil
}

func (a *Accrual) Close() {
	a.Orders.Close()
	a.Rules.Close()
}
