package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/worker"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"go.uber.org/zap"
)

const (
	MsgEmptyMatch        = "non-empty match is required"
	MsgInvalidReward     = "reward must be greater than 0"
	MsgInvalidRewardType = "reward_type must be 'pt' or '%'"
)

type reqRewardRule struct {
	Match      string  `json:"match"`
	Reward     float64 `json:"reward"`
	RewardType string  `json:"reward_type"`
}

func (r reqRewardRule) Valid() validator.Problems {
	problems := make(validator.Problems)

	if r.Match == "" {
		problems["match"] = MsgEmptyMatch
	}

	if r.Reward <= 0 {
		problems["reward"] = MsgInvalidReward
	}

	if r.RewardType != "pt" && r.RewardType != "%" {
		problems["reward_type"] = MsgInvalidRewardType
	}

	return problems
}

type RuleRegisterer interface {
	RegisterRule(ctx context.Context, rule *model.RewardRule) error
}

func prepareRule(rule reqRewardRule) (*model.RewardRule, error) {
	mr, err := model.NewRewardRule(rule.Match, rule.Reward, model.RewardType(rule.RewardType))
	if err != nil {
		return nil, fmt.Errorf("create model RewardRule: %w", err)
	}
	return mr, nil
}

func handleRewardRule(l *zap.Logger, reg RuleRegisterer, inv worker.RulesProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rule, err := codec.Decode[reqRewardRule](r)
		if err != nil {
			l.Debug("decode json request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = validator.IsValid(rule)
		if err != nil {
			l.Debug("check validity", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		preparedRule, err := prepareRule(rule)
		if err != nil {
			l.Error("prepare model RewardRule for register", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		if err := reg.RegisterRule(context.Background(), preparedRule); err != nil {
			if errors.Is(err, service.ErrRuleAlreadyRegistered) {
				w.WriteHeader(http.StatusConflict)
				return
			}

			l.Error("register reward rule via service", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		inv.MarkDirty()

		w.WriteHeader(http.StatusOK)
	}
}
