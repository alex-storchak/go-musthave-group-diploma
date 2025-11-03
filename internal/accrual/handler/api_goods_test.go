package handler

import (
	handlermocks "github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/handler/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	workermocks "github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/worker/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRewardRule_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()

	mockRegisterer := handlermocks.NewMockRuleRegisterer(t)
	mockRegisterer.EXPECT().
		RegisterRule(mock.Anything, mock.AnythingOfType("*model.RewardRule")).
		Return(nil)

	mockProvider := workermocks.NewMockRulesProvider(t)
	mockProvider.EXPECT().
		MarkDirty().
		Once()

	handler := handleRewardRule(logger, mockRegisterer, mockProvider)

	// Execute
	reqBody := `{"match": "test", "reward": 10, "reward_type": "pt"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRegisterer.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestHandleRewardRule_ValidationError(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRegisterer := handlermocks.NewMockRuleRegisterer(t)
	mockProvider := workermocks.NewMockRulesProvider(t)

	handler := handleRewardRule(logger, mockRegisterer, mockProvider)

	// Execute
	// Invalid request - missing required fields
	reqBody := `{"match": "", "reward": 0, "reward_type": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRegisterer.AssertNotCalled(t, "RegisterRule")
	mockProvider.AssertNotCalled(t, "MarkDirty")
}

func TestHandleRewardRule_Conflict(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRegisterer := handlermocks.NewMockRuleRegisterer(t)
	mockRegisterer.EXPECT().
		RegisterRule(mock.Anything, mock.AnythingOfType("*model.RewardRule")).
		Return(service.ErrRuleAlreadyRegistered)
	mockProvider := workermocks.NewMockRulesProvider(t)

	handler := handleRewardRule(logger, mockRegisterer, mockProvider)

	// Execute
	reqBody := `{"match": "test", "reward": 10, "reward_type": "pt"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)
	mockRegisterer.AssertExpectations(t)
	mockProvider.AssertNotCalled(t, "MarkDirty")
}

func TestReqRewardRule_Valid(t *testing.T) {
	tests := []struct {
		name     string
		rule     reqRewardRule
		expected validator.Problems
	}{
		{
			name:     "valid rule",
			rule:     reqRewardRule{Match: "test", Reward: 10, RewardType: "%"},
			expected: validator.Problems{},
		},
		{
			name:     "empty match",
			rule:     reqRewardRule{Match: "", Reward: 10, RewardType: "pt"},
			expected: validator.Problems{"match": MsgEmptyMatch},
		},
		{
			name:     "invalid reward",
			rule:     reqRewardRule{Match: "test", Reward: 0, RewardType: "pt"},
			expected: validator.Problems{"reward": MsgInvalidReward},
		},
		{
			name:     "invalid reward type",
			rule:     reqRewardRule{Match: "test", Reward: 10, RewardType: "invalid"},
			expected: validator.Problems{"reward_type": MsgInvalidRewardType},
		},
		{
			name: "multiple errors",
			rule: reqRewardRule{Match: "", Reward: 0, RewardType: "invalid"},
			expected: validator.Problems{
				"match":       MsgEmptyMatch,
				"reward":      MsgInvalidReward,
				"reward_type": MsgInvalidRewardType,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := tt.rule.Valid()
			assert.Equal(t, tt.expected, problems)
		})
	}
}
