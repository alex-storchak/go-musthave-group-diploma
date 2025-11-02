package withdrawals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	utils "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils/auth"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const defaultCtxTimeout = 60 * time.Second

type Gophermart interface {
	StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal, setDefaultBalance models.SetDefaultBalanceRequest) error
	IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) ([]models.IndexWithdrawalResponse, error)
	CountWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (int64, error)
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

		indexWithdrawal := models.IndexWithdrawal{UserID: userID}

		// Устанавливаем таймаут контекста
		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		// Проверяем количество записей
		count, err := gophermart.CountWithdrawal(ctx, indexWithdrawal)
		if err != nil {
			logger.Error("count withdrawal", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if count < 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Получаем все выводы сразу (синхронно)
		withdrawals, err := gophermart.IndexWithdrawal(ctx, indexWithdrawal)
		if err != nil {
			logger.Error("index withdrawal", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Формируем JSON‑ответ
		response, err := json.Marshal(withdrawals)
		if err != nil {
			logger.Error("marshal withdrawals to json", zap.Error(err))
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

func getStoreWrapper(w http.ResponseWriter, r *http.Request, logger *zap.Logger) (*validators.StoreWithdrawalWrapper, error) {
	storeWithdrawalWrapper := &validators.StoreWithdrawalWrapper{
		StoreWithdrawalRequest: &models.StoreWithdrawalRequest{},
	}

	problems, err := validators.Decode(r, storeWithdrawalWrapper)
	if err != nil {
		logger.Debug("bad request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("bad request %w", err)
	}

	if len(problems) > 0 {
		logger.Debug("bad request", zap.String("problems", fmt.Sprintf("%v", problems)))
		myerrors.ErrorValidateJSONResponse(w, problems, http.StatusBadRequest)
		return nil, fmt.Errorf("bad request %w: problems: %v", myerrors.ErrValidation, problems)
	}

	return storeWithdrawalWrapper, nil
}

func Store(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		storeWithdrawalWrapper, err := getStoreWrapper(w, r, logger)
		if err != nil {
			return
		}

		userID, err := utils.GetCtxUserID(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err = gophermart.StoreWithdrawal(r.Context(), models.StoreWithdrawal{
			Number: storeWithdrawalWrapper.Number,
			Sum:    storeWithdrawalWrapper.Sum,
			UserID: userID,
		}, models.SetDefaultBalanceRequest{
			UserID:         userID,
			Current:        0,
			TotalWithdrawn: 0,
		})

		if err != nil {
			if errors.Is(err, myerrors.ErrBalance) {
				logger.Debug("balance", zap.Error(err))
				w.WriteHeader(http.StatusPaymentRequired)
				return
			}

			if errors.Is(err, myerrors.ErrOrderNotFound) {
				logger.Debug("order", zap.Error(err))
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}

			logger.Error("storing withdrawal", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
