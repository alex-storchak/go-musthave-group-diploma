package handler

import (
	"context"
	"encoding/json"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/handler/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleOrderNumber(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		orderNumber    string
		setupMock      func(m *mocks.MockOrderInformer)
		expectedStatus int
		expectedJSON   string
		description    string
	}{
		{
			name:        "successful order retrieval",
			orderNumber: "4561261212345467",
			setupMock: func(m *mocks.MockOrderInformer) {
				m.EXPECT().
					InformOrder(mock.Anything, "4561261212345467").
					Return(&model.Order{
						Number:  "4561261212345467",
						Status:  model.StatusProcessed,
						Accrual: 100,
					}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedJSON: `{
                "order": "4561261212345467",
                "status": "PROCESSED",
                "accrual": 100
            }`,
			description: "should return order with 200 (Ok) status",
		},
		{
			name:        "order not found",
			orderNumber: "nonexistent123",
			setupMock: func(m *mocks.MockOrderInformer) {
				m.EXPECT().
					InformOrder(mock.Anything, "nonexistent123").
					Return(nil, repository.ErrOrderNotFound).
					Once()
			},
			expectedStatus: http.StatusNoContent,
			expectedJSON:   "",
			description:    "should return 204 (No content) for non-existent order",
		},
		{
			name:        "internal server error",
			orderNumber: "4561261212345467",
			setupMock: func(m *mocks.MockOrderInformer) {
				m.EXPECT().
					InformOrder(mock.Anything, "4561261212345467").
					Return(nil, assert.AnError).
					Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedJSON:   "",
			description:    "should return 500 (Internal server error) on internal error",
		},
		{
			name:        "order with processing status",
			orderNumber: "1234567890",
			setupMock: func(m *mocks.MockOrderInformer) {
				m.EXPECT().
					InformOrder(mock.Anything, "1234567890").
					Return(&model.Order{
						Number: "1234567890",
						Status: model.StatusProcessing,
					}, nil).
					Once()
			},
			expectedStatus: http.StatusOK,
			expectedJSON: `{
                "order": "1234567890", 
                "status": "PROCESSING"
            }`,
			description: "should return order with processing status",
		},
		{
			name:        "order with invalid status",
			orderNumber: "1111111111",
			setupMock: func(m *mocks.MockOrderInformer) {
				m.EXPECT().
					InformOrder(mock.Anything, "1111111111").
					Return(&model.Order{
						Number: "1111111111",
						Status: model.StatusInvalid,
					}, nil).
					Once()
			},
			expectedStatus: http.StatusOK,
			expectedJSON: `{
                "order": "1111111111",
                "status": "INVALID" 
            }`,
			description: "should return order with 200 (Ok) status when invalid order number passed",
		},
		{
			name:        "order without accrual",
			orderNumber: "2222222222",
			setupMock: func(m *mocks.MockOrderInformer) {
				m.EXPECT().
					InformOrder(mock.Anything, "2222222222").
					Return(&model.Order{
						Number: "2222222222",
						Status: model.StatusRegistered,
					}, nil).
					Once()
			},
			expectedStatus: http.StatusOK,
			expectedJSON: `{
                "order": "2222222222",
                "status": "REGISTERED"
            }`,
			description: "should return order without accrual field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockInf := &mocks.MockOrderInformer{}
			tt.setupMock(mockInf)

			handler := handleOrderNumber(logger, mockInf)

			// Execute
			req := httptest.NewRequest("GET", "/orders/"+tt.orderNumber, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("number", tt.orderNumber)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			handler(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)

			if tt.expectedStatus == http.StatusOK && tt.expectedJSON != "" {
				// Нормализуем JSON для сравнения
				var expected map[string]interface{}
				err := json.Unmarshal([]byte(tt.expectedJSON), &expected)
				assert.NoError(t, err)

				var actual map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &actual)
				assert.NoError(t, err)

				assert.Equal(t, expected, actual)
			} else if tt.expectedStatus == http.StatusNoContent {
				assert.Empty(t, w.Body.Bytes())
			}
		})
	}
}
