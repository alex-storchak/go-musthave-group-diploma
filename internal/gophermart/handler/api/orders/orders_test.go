package orders

import (
	ordersmocks "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/orders/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIndex_Unauthorized(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	req := httptest.NewRequest(http.MethodGet, "/user/orders", nil)

	w := httptest.NewRecorder()
	handler := Index(logger, gophermart)

	handler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestIndex_NoContent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	gophermart.On("CountOrder", mock.Anything, models.IndexOrder{UserID: models.UserID(1)}).
		Return(int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/user/orders", nil)
	ctx := auth.WithUser(t.Context(), models.UserID(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler := Index(logger, gophermart)
	handler(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	gophermart.AssertExpectations(t)
}

func TestIndex_StreamSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	gophermart.On("CountOrder", mock.Anything, models.IndexOrder{UserID: models.UserID(1)}).
		Return(int64(2), nil)

	ordersChan := make(chan models.IndexOrderResponse, 2)
	errChan := make(chan error, 1)

	timeNow := time.Now()
	ordersChan <- models.IndexOrderResponse{
		Number:     "9278923470",
		Status:     string(models.OrderProcessed),
		Accrual:    models.RoundedFloat64(500),
		UploadedAt: &timeNow,
	}
	ordersChan <- models.IndexOrderResponse{
		Number:     "12345678903",
		Status:     string(models.OrderProcessing),
		UploadedAt: &timeNow,
	}

	close(ordersChan)
	close(errChan)

	gophermart.On("IndexOrder", mock.Anything, models.IndexOrder{UserID: models.UserID(1)}).
		Return(
			(<-chan models.IndexOrderResponse)(ordersChan),
			(<-chan error)(errChan),
		)

	req := httptest.NewRequest(http.MethodGet, "/user/orders", nil)
	ctx := auth.WithUser(t.Context(), models.UserID(1))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler := Index(logger, gophermart)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	gophermart.AssertExpectations(t)
}

func TestStore_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	gophermart.On("StoreOrder", mock.Anything, mock.AnythingOfType("models.StoreOrder")).
		Return(nil)

	handler := Store(logger, gophermart)

	reqBody := `12345678903`
	req := httptest.NewRequest(http.MethodPost, "/user/orders", strings.NewReader(reqBody))
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	gophermart.AssertExpectations(t)
}

func TestStore_Unauthorized(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	handler := Store(logger, gophermart)

	reqBody := `12345678903`
	req := httptest.NewRequest(http.MethodPost, "/user/orders", strings.NewReader(reqBody))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	gophermart.AssertNotCalled(t, "StoreOrder") // Метод не должен вызываться
}

func TestStore_ValidationError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	// Мокируем ошибку валидации
	gophermart.On("StoreOrder", mock.Anything, mock.AnythingOfType("models.StoreOrder")).
		Return(nil)

	handler := Store(logger, gophermart)

	reqBody := `12345678904` // Невалидный номер
	req := httptest.NewRequest(http.MethodPost, "/user/orders", strings.NewReader(reqBody))
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	gophermart.AssertNotCalled(t, "StoreOrder")
}

func TestStore_StoreError_Conflict(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	// Ошибка конфликта — статус должен быть 200 (по логике вашего приложения)
	gophermart.On("StoreOrder", mock.Anything, mock.AnythingOfType("models.StoreOrder")).
		Return(myerrors.ErrConflictNumber)

	gophermart.On("GetOrder", mock.Anything, mock.AnythingOfType("models.GetOrder")).
		Return(&models.Order{
			Number: "12345678903",
			UserID: models.UserID(1),
		}, nil)

	handler := Store(logger, gophermart)

	reqBody := `12345678903`
	req := httptest.NewRequest(http.MethodPost, "/store", strings.NewReader(reqBody))
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Предполагаем, что при конфликте возвращается 200
	gophermart.AssertExpectations(t)
}

func TestStore_StoreError_Conflict2(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	gophermart := &ordersmocks.MockGophermart{}

	// Ошибка конфликта — статус должен быть 200 (по логике вашего приложения)
	gophermart.On("StoreOrder", mock.Anything, mock.AnythingOfType("models.StoreOrder")).
		Return(myerrors.ErrConflictNumber)

	gophermart.On("GetOrder", mock.Anything, mock.Anything).
		Return(&models.Order{
			Number: "12345678903",
			UserID: models.UserID(2),
		}, nil)

	handler := Store(logger, gophermart)

	reqBody := `12345678903`
	req := httptest.NewRequest(http.MethodPost, "/store", strings.NewReader(reqBody))
	req = req.WithContext(auth.WithUser(t.Context(), models.UserID(1)))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusConflict, w.Code) // Предполагаем, что при конфликте возвращается 200
	gophermart.AssertExpectations(t)
}
