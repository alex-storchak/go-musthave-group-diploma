package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

var (
	ErrOrderAlreadyRegistered = errors.New("order already registered")
	ErrRuleAlreadyRegistered  = errors.New("rule already registered")
)

type OrdersRepository interface {
	Add(ctx context.Context, order *model.Order) error
	Has(ctx context.Context, number string) (bool, error)
	Get(ctx context.Context, number string) (*model.Order, error)
	Close()
}

type RulesRepository interface {
	Add(ctx context.Context, rule *model.RewardRule) error
	Has(ctx context.Context, match string) (bool, error)
	All(ctx context.Context) ([]model.RewardRule, error)
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

func (a *Accrual) InformOrder(ctx context.Context, number string) (*model.Order, error) {
	order, err := a.Orders.Get(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("get order from repo: %w", err)
	}
	return order, nil
}

func (a *Accrual) ProcessOrder(order *model.Order, rules []model.RewardRule) {
	totalAccrual := 0.0

	for _, g := range order.Goods {
		accrual := a.calculateGoodAccrual(g, rules, order.RegisteredAt)
		totalAccrual += accrual
	}

	if totalAccrual == 0 {
		a.logger.Info("no accrual for order", zap.String("order_number", order.Number))
		order.Status = repository.StatusInvalid
		return
	}

	order.Accrual = totalAccrual
	order.Status = repository.StatusProcessed
}

func (a *Accrual) calculateGoodAccrual(
	good model.Good,
	rules []model.RewardRule,
	orderRegisteredAt pgtype.Timestamptz,
) float64 {
	accrual := 0.0
	description := strings.ToLower(good.Description)

	for _, rule := range rules {
		ruleTime := rule.CreatedAt.Time
		orderTime := orderRegisteredAt.Time
		isApplicable := ruleTime.Before(orderTime)
		if !isApplicable {
			continue
		}
		if strings.Contains(description, strings.ToLower(rule.Match)) {
			switch rule.RewardType {
			case model.Percent:
				accrual += good.Price * rule.Reward / 100
			case model.Points:
				accrual += rule.Reward
			}
		}
	}

	return accrual
}

func (a *Accrual) Close() {
	a.Orders.Close()
	a.Rules.Close()
}
