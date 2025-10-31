package orders

import (
	"context"
	"encoding/json"
	"errors"
	utils "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Gophermart interface {
	CountOrder(ctx context.Context, indexOrder models.IndexOrder) (int64, error)
	IndexOrder(ctx context.Context, indexOrder models.IndexOrder) ([]models.IndexOrderResponse, error)
	StoreOrder(ctx context.Context, storeOrder models.StoreOrder) error
	GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error)
}

func Index(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Получаем ID пользователя из контекста
		userID, err := utils.GetCtxUserID(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		indexOrder := models.IndexOrder{UserID: userID}

		// Устанавливаем таймаут контекста
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		// Проверяем количество записей
		count, err := gophermart.CountOrder(ctx, indexOrder)
		if err != nil {
			logger.Error("count order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if count < 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Получаем все заказы сразу (синхронно)
		orders, err := gophermart.IndexOrder(ctx, indexOrder) // Теперь возвращает []models.IndexOrderResponse, error
		if err != nil {
			logger.Error("index order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Формируем JSON-ответ
		response, err := json.Marshal(orders)
		if err != nil {
			logger.Error("marshal orders to json", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Отправляем ответ
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)
		if err != nil {
			logger.Error("write response", zap.Error(err))
		}
	}
}

func Store(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := utils.GetCtxUserID(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		storeOrderWrapper := &validators.StoreOrderWrapper{
			StoreOrder: &models.StoreOrder{},
		}

		problems, err := validators.Decode(r, storeOrderWrapper)
		if err != nil {
			logger.Debug("bad request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(problems) > 0 {
			status := http.StatusBadRequest

			if numberProblems, ok := problems["number"]; ok {
				if _, hasNoValidError := numberProblems["is_valid"]; hasNoValidError {
					status = http.StatusUnprocessableEntity
				}
			}
			problemsStr := validators.ValidErrToStr(problems)
			logger.Debug("bad request", zap.String("problems", problemsStr))
			w.WriteHeader(status)
			w.Write([]byte(problemsStr))
			return
		}
		storeOrderWrapper.StoreOrder.UserID = userID
		err = gophermart.StoreOrder(r.Context(), *storeOrderWrapper.StoreOrder)
		isErrConflictNumber := errors.Is(err, myerrors.ErrConflictNumber)
		if err != nil && !isErrConflictNumber {
			logger.Error("error storing order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		status := http.StatusAccepted

		if isErrConflictNumber {
			order, err := gophermart.GetOrder(r.Context(), models.GetOrder{
				Number: storeOrderWrapper.StoreOrder.Number,
			})
			if err != nil {
				logger.Error("error get order", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if order == nil {
				logger.Error("no get order", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if order.UserID == storeOrderWrapper.StoreOrder.UserID {
				status = http.StatusOK
			} else {
				status = http.StatusConflict
			}

		}

		w.WriteHeader(status)
	}
}
