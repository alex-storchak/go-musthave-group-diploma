package handler

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/handler/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReqOrder_Valid(t *testing.T) {
	tests := []struct {
		name     string
		order    reqOrder
		expected validator.Problems
	}{
		{
			name: "valid order",
			order: reqOrder{
				Number: "4561261212345467", // Valid Luhn
				Goods: []reqGood{
					{Description: "Valid", Price: 1500.50},
					{Description: "Valid", Price: 25.99},
				},
			},
			expected: validator.Problems{},
		},
		{
			name:     "empty order number",
			order:    reqOrder{Number: "", Goods: []reqGood{{Description: "Test", Price: 10}}},
			expected: validator.Problems{"order": MsgEmptyOrderNumber},
		},
		{
			name:     "invalid Luhn number",
			order:    reqOrder{Number: "1234567890", Goods: []reqGood{{Description: "Test", Price: 10}}},
			expected: validator.Problems{"order": MsgInvalidOrderNumber},
		},
		{
			name: "empty goods description",
			order: reqOrder{
				Number: "4561261212345467",
				Goods: []reqGood{
					{Description: "", Price: 10},
					{Description: "Valid", Price: 20},
				},
			},
			expected: validator.Problems{"goods.0.description": MsgEmptyGoodDescription},
		},
		{
			name: "invalid goods price",
			order: reqOrder{
				Number: "4561261212345467",
				Goods: []reqGood{
					{Description: "Valid", Price: 0},
					{Description: "Valid", Price: -5},
				},
			},
			expected: validator.Problems{
				"goods.0.price": MsgInvalidGoodPrice,
				"goods.1.price": MsgInvalidGoodPrice,
			},
		},
		{
			name: "multiple validation errors",
			order: reqOrder{
				Number: "123", // Invalid Luhn
				Goods: []reqGood{
					{Description: "", Price: 0},
				},
			},
			expected: validator.Problems{
				"order":               MsgInvalidOrderNumber,
				"goods.0.description": MsgEmptyGoodDescription,
				"goods.0.price":       MsgInvalidGoodPrice,
			},
		},
		{
			name:     "no goods",
			order:    reqOrder{Number: "4561261212345467", Goods: []reqGood{}},
			expected: validator.Problems{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := tt.order.Valid()
			assert.Equal(t, tt.expected, problems)
		})
	}
}

func TestPrepareOrder(t *testing.T) {
	tests := []struct {
		name     string
		input    reqOrder
		expected *model.Order
	}{
		{
			name: "basic order",
			input: reqOrder{
				Number: "4561 2612 1234 5467",
				Goods: []reqGood{
					{Description: "Good 1", Price: 1500},
					{Description: "Good 2", Price: 25},
				},
			},
			expected: &model.Order{
				Number: "4561261212345467", // whitespaces removed
				Goods: []model.Good{
					{Description: "Good 1", Price: 1500},
					{Description: "Good 2", Price: 25},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareOrder(tt.input)
			assert.Equal(t, tt.expected.Number, result.Number)
			assert.Equal(t, len(tt.expected.Goods), len(result.Goods))

			for i, expectedGood := range tt.expected.Goods {
				assert.Equal(t, expectedGood, result.Goods[i])
			}
		})
	}
}

func TestHandleOrders(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(mockReg *mocks.MockOrderRegisterer)
		expectedStatus int
		description    string
	}{
		{
			name: "successful order registration",
			requestBody: `{
                "order": "4561261212345467",
                "goods": [
                    {"description": "Good 1", "price": 1500},
                    {"description": "Good 2", "price": 25}
                ]
            }`,
			setupMock: func(m *mocks.MockOrderRegisterer) {
				m.EXPECT().
					RegisterOrder(mock.Anything, mock.AnythingOfType("*model.Order")).
					Return(nil).
					Once()
			},
			expectedStatus: http.StatusAccepted,
			description:    "should return 202 (Accepted) on valid order",
		},
		{
			name:           "invalid JSON",
			requestBody:    `invalid json`,
			setupMock:      func(m *mocks.MockOrderRegisterer) {},
			expectedStatus: http.StatusBadRequest,
			description:    "should reject invalid JSON",
		},
		{
			name: "request validation failed",
			requestBody: `{
                "order": "",
                "goods": [{"description": "", "price": 0}]
            }`,
			setupMock:      func(m *mocks.MockOrderRegisterer) {},
			expectedStatus: http.StatusBadRequest,
			description:    "should reject order that doesn't pass request validation",
		},
		{
			name: "order already exists",
			requestBody: `{
                "order": "4561261212345467",
                "goods": [{"description": "Test", "price": 10}]
            }`,
			setupMock: func(m *mocks.MockOrderRegisterer) {
				m.EXPECT().
					RegisterOrder(mock.Anything, mock.AnythingOfType("*model.Order")).
					Return(service.ErrOrderAlreadyRegistered).
					Once()
			},
			expectedStatus: http.StatusConflict,
			description:    "should return 409 (status conflict) for duplicate order",
		},
		{
			name: "internal server error",
			requestBody: `{
                "order": "4561261212345467",
                "goods": [{"description": "Test", "price": 10}]
            }`,
			setupMock: func(m *mocks.MockOrderRegisterer) {
				m.EXPECT().
					RegisterOrder(mock.Anything, mock.AnythingOfType("*model.Order")).
					Return(assert.AnError).
					Once()
			},
			expectedStatus: http.StatusInternalServerError,
			description:    "should return status 500 (internal server error) on internal error",
		},
		{
			name: "empty goods list",
			requestBody: `{
                "order": "4561261212345467",
                "goods": []
            }`,
			setupMock: func(m *mocks.MockOrderRegisterer) {
				m.EXPECT().
					RegisterOrder(mock.Anything, mock.AnythingOfType("*model.Order")).
					Return(nil).
					Once()
			},
			expectedStatus: http.StatusAccepted,
			description:    "should return 202 (Accepted) on order with empty goods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockReg := mocks.NewMockOrderRegisterer(t)
			tt.setupMock(mockReg)

			handler := handleOrders(logger, mockReg)

			// Execute
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}
