package service

import (
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service/mocks"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var errRandom = errors.New("random error")

//nolint:dupl // order and rule are different entities with possible registration logic
func TestAccrual_RegisterOrder(t *testing.T) {
	tests := []struct {
		name        string
		order       *model.Order
		hasOrderErr error
		hasOrder    bool
		addOrderErr error
		expectedErr error
	}{
		{
			name:        "successful registration",
			order:       &model.Order{Number: "12345"},
			hasOrder:    false,
			expectedErr: nil,
		},
		{
			name:        "order already exists",
			order:       &model.Order{Number: "12345"},
			hasOrder:    true,
			expectedErr: ErrOrderAlreadyRegistered,
		},
		{
			name:        "error checking order existence",
			order:       &model.Order{Number: "12345"},
			hasOrderErr: errRandom,
			expectedErr: errRandom,
		},
		{
			name:        "error adding order",
			order:       &model.Order{Number: "12345"},
			hasOrder:    false,
			addOrderErr: errRandom,
			expectedErr: errRandom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx := t.Context()
			mockOrdersRepo := mocks.NewMockOrdersRepository(t)
			mockRulesRepo := mocks.NewMockRulesRepository(t)
			logger := zap.NewNop()

			accrualService := NewAccrual(mockOrdersRepo, mockRulesRepo, logger)

			mockOrdersRepo.EXPECT().
				Has(ctx, tt.order.Number).
				Return(tt.hasOrder, tt.hasOrderErr)

			if tt.hasOrderErr == nil && !tt.hasOrder {
				mockOrdersRepo.EXPECT().
					Add(ctx, tt.order).
					Return(tt.addOrderErr)
			}

			// Execute
			err := accrualService.RegisterOrder(ctx, tt.order)

			// Assert
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:dupl // order and rule are different entities with possible registration logic
func TestAccrual_RegisterRule(t *testing.T) {
	tests := []struct {
		name        string
		rule        *model.RewardRule
		hasRuleErr  error
		hasRule     bool
		addRuleErr  error
		expectedErr error
	}{
		{
			name:        "successful registration",
			rule:        &model.RewardRule{Match: "test"},
			hasRule:     false,
			expectedErr: nil,
		},
		{
			name:        "rule already exists",
			rule:        &model.RewardRule{Match: "test"},
			hasRule:     true,
			expectedErr: ErrRuleAlreadyRegistered,
		},
		{
			name:        "error checking rule existence",
			rule:        &model.RewardRule{Match: "test"},
			hasRuleErr:  errRandom,
			expectedErr: errRandom,
		},
		{
			name:        "error adding rule",
			rule:        &model.RewardRule{Match: "test"},
			hasRule:     false,
			addRuleErr:  errRandom,
			expectedErr: errRandom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx := t.Context()
			mockOrdersRepo := mocks.NewMockOrdersRepository(t)
			mockRulesRepo := mocks.NewMockRulesRepository(t)
			logger := zap.NewNop()

			accrualService := NewAccrual(mockOrdersRepo, mockRulesRepo, logger)

			mockRulesRepo.EXPECT().
				Has(ctx, tt.rule.Match).
				Return(tt.hasRule, tt.hasRuleErr).
				Once()

			if tt.hasRuleErr == nil && !tt.hasRule {
				mockRulesRepo.EXPECT().
					Add(ctx, tt.rule).
					Return(tt.addRuleErr).
					Once()
			}

			// Execute
			err := accrualService.RegisterRule(ctx, tt.rule)

			// Assert
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAccrual_InformOrder(t *testing.T) {
	tests := []struct {
		name          string
		orderNumber   string
		getOrder      *model.Order
		getOrderErr   error
		expectedOrder *model.Order
		expectedErr   error
	}{
		{
			name:        "successful get order",
			orderNumber: "12345",
			getOrder: &model.Order{
				Number:  "12345",
				Status:  model.StatusProcessed,
				Accrual: 100,
			},
			expectedOrder: &model.Order{
				Number:  "12345",
				Status:  model.StatusProcessed,
				Accrual: 100,
			},
			expectedErr: nil,
		},
		{
			name:          "order not found",
			orderNumber:   "12345",
			getOrder:      nil,
			getOrderErr:   repository.ErrOrderNotFound,
			expectedOrder: nil,
			expectedErr:   repository.ErrOrderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx := t.Context()
			mockOrdersRepo := mocks.NewMockOrdersRepository(t)
			mockRulesRepo := mocks.NewMockRulesRepository(t)
			logger := zap.NewNop()

			accrualService := NewAccrual(mockOrdersRepo, mockRulesRepo, logger)

			mockOrdersRepo.EXPECT().
				Get(ctx, tt.orderNumber).
				Return(tt.getOrder, tt.getOrderErr).
				Once()

			// Execute
			order, err := accrualService.InformOrder(ctx, tt.orderNumber)

			// Assert
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				assert.Nil(t, order)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOrder, order)
			}
		})
	}
}

func TestAccrual_ProcessOrder(t *testing.T) {
	now := time.Now()
	registeredAt := pgtype.Timestamptz{Time: now, Valid: true}
	ruleCreatedAtBefore := pgtype.Timestamptz{Time: now.Add(-time.Hour), Valid: true}
	ruleCreatedAtAfter := pgtype.Timestamptz{Time: now.Add(time.Hour), Valid: true}

	tests := []struct {
		name            string
		order           *model.Order
		rules           []model.RewardRule
		expectedStatus  string
		expectedAccrual float64
	}{
		{
			name: "no matching rules - status invalid",
			order: &model.Order{
				Number: "12345",
				Goods: model.Goods{
					{Description: "Good 1", Price: 100.0},
					{Description: "Good 2", Price: 200.0},
				},
				RegisteredAt: registeredAt,
			},
			rules: []model.RewardRule{
				{
					Match:      "non-match",
					Reward:     10.0,
					RewardType: model.Percent,
					CreatedAt:  ruleCreatedAtBefore,
				},
			},
			expectedStatus:  model.StatusInvalid,
			expectedAccrual: 0.0,
		},
		{
			name: "percent reward calculation",
			order: &model.Order{
				Number: "12345",
				Goods: model.Goods{
					{Description: "Good 1", Price: 100.0},
					{Description: "Good 2", Price: 200.0},
				},
				RegisteredAt: registeredAt,
			},
			rules: []model.RewardRule{
				{
					Match:      "Good 1",
					Reward:     10.0,
					RewardType: model.Percent,
					CreatedAt:  ruleCreatedAtBefore,
				},
			},
			expectedStatus:  model.StatusProcessed,
			expectedAccrual: 10.0,
		},
		{
			name: "points reward calculation",
			order: &model.Order{
				Number: "12345",
				Goods: model.Goods{
					{Description: "Good 1", Price: 50.0},
					{Description: "Good 2", Price: 100.0},
				},
				RegisteredAt: registeredAt,
			},
			rules: []model.RewardRule{
				{
					Match:      "Good 1",
					Reward:     25.0,
					RewardType: model.Points,
					CreatedAt:  ruleCreatedAtBefore,
				},
			},
			expectedStatus:  model.StatusProcessed,
			expectedAccrual: 25.0,
		},
		{
			name: "multiple rules and goods",
			order: &model.Order{
				Number: "12345",
				Goods: model.Goods{
					{Description: "Percent Good 1", Price: 200.0},
					{Description: "Points Good 2", Price: 100.0},
					{Description: "Regular Good 3", Price: 50.0},
				},
				RegisteredAt: registeredAt,
			},
			rules: []model.RewardRule{
				{
					Match:      "Good 1",
					Reward:     15.0,
					RewardType: model.Percent,
					CreatedAt:  ruleCreatedAtBefore,
				},
				{
					Match:      "Good 2",
					Reward:     50.0,
					RewardType: model.Points,
					CreatedAt:  ruleCreatedAtBefore,
				},
			},
			expectedStatus:  model.StatusProcessed,
			expectedAccrual: 80.0,
		},
		{
			name: "rule created after order - not applicable",
			order: &model.Order{
				Number: "12345",
				Goods: model.Goods{
					{Description: "Future Good 1", Price: 100.0},
				},
				RegisteredAt: registeredAt,
			},
			rules: []model.RewardRule{
				{
					Match:      "Good 1",
					Reward:     10.0,
					RewardType: model.Percent,
					CreatedAt:  ruleCreatedAtAfter,
				},
			},
			expectedStatus:  model.StatusInvalid,
			expectedAccrual: 0.0,
		},
		{
			name: "case insensitive matching",
			order: &model.Order{
				Number: "12345",
				Goods: model.Goods{
					{Description: "Good 1", Price: 100.0},
				},
				RegisteredAt: registeredAt,
			},
			rules: []model.RewardRule{
				{
					Match:      "GOOD 1",
					Reward:     10.0,
					RewardType: model.Percent,
					CreatedAt:  ruleCreatedAtBefore,
				},
			},
			expectedStatus:  model.StatusProcessed,
			expectedAccrual: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockOrdersRepo := mocks.NewMockOrdersRepository(t)
			mockRulesRepo := mocks.NewMockRulesRepository(t)
			logger := zap.NewNop()

			accrualService := NewAccrual(mockOrdersRepo, mockRulesRepo, logger)

			// Execute
			ttOrder := *tt.order // avoid modifying test data
			accrualService.ProcessOrder(&ttOrder, tt.rules)

			// Assert
			assert.Equal(t, tt.expectedStatus, ttOrder.Status)
			assert.Equal(t, tt.expectedAccrual, ttOrder.Accrual)
		})
	}
}

func TestAccrual_Close(t *testing.T) {
	mockOrdersRepo := mocks.NewMockOrdersRepository(t)
	mockRulesRepo := mocks.NewMockRulesRepository(t)
	logger := zap.NewNop()

	accrualService := NewAccrual(mockOrdersRepo, mockRulesRepo, logger)

	mockOrdersRepo.EXPECT().Close().Once()
	mockRulesRepo.EXPECT().Close().Once()

	accrualService.Close()
}
