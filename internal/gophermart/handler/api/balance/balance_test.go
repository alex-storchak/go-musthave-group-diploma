package balance

import (
	balancemocks "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/balance/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShow_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &balancemocks.MockGophermart{}

	expectedBalance := &models.ShowBalanceResponse{
		Current:        100.50,
		TotalWithdrawn: 20.00,
	}
	gophermart.On("GetBalance", mock.Anything, models.GetBalanceRequest{UserID: models.UserID(1)}).
		Return(expectedBalance, nil)

	handler := Show(logger, gophermart)

	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	gophermart.AssertExpectations(t)
}

func TestShow_Unauthorized_NoUserID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &balancemocks.MockGophermart{}

	handler := Show(logger, gophermart)

	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	// Нет userID в контексте

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	gophermart.AssertNotCalled(t, "GetBalance")
	gophermart.AssertNotCalled(t, "CreateDefaultBalance")
}
