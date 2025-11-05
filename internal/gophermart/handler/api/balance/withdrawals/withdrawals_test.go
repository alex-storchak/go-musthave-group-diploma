package withdrawals

import (
	withdrawalmocks "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance/withdrawals/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIndex_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &withdrawalmocks.MockGophermart{}

	gophermart.On("CountWithdrawal", mock.Anything, models.IndexWithdrawal{UserID: models.UserID(1)}).
		Return(int64(2), nil)

	timeNow := time.Now()
	expectedWithdrawals := []models.IndexWithdrawalResponse{
		{
			Number:      "2377225624",
			Sum:         100.50,
			ProcessedAt: &timeNow,
		},
		{
			Number:      "346436439",
			Sum:         500,
			ProcessedAt: &timeNow,
		},
	}
	gophermart.On("IndexWithdrawal", mock.Anything, models.IndexWithdrawal{UserID: models.UserID(1)}).
		Return(expectedWithdrawals, nil)

	handler := Index(logger, gophermart)

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	gophermart.AssertExpectations(t)
}

func TestIndex_Unauthorized_NoUserID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &withdrawalmocks.MockGophermart{}

	handler := Index(logger, gophermart)

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	// Нет userID в контексте

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	gophermart.AssertNotCalled(t, "CountWithdrawal")
	gophermart.AssertNotCalled(t, "IndexWithdrawal")
}

func TestIndex_NoContent_ZeroCount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &withdrawalmocks.MockGophermart{}

	// CountWithdrawal → 0 записей
	gophermart.On("CountWithdrawal", mock.Anything, models.IndexWithdrawal{UserID: models.UserID(1)}).
		Return(int64(0), nil)

	handler := Index(logger, gophermart)

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String()) // Тело должно быть пустым
	gophermart.AssertExpectations(t)
	gophermart.AssertNotCalled(t, "IndexWithdrawal") // Не должен вызываться
}
