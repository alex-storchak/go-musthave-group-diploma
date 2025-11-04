package orders

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
	CountOrder(ctx context.Context, indexOrder models.IndexOrder) (int64, error)
	IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.IndexOrderResponse, <-chan error)
	StoreOrder(ctx context.Context, storeOrder models.StoreOrder) error
	GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error)
}

func Index(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := prepareResponseWriter(w); err != nil {
			logger.Error("failed to set response headers", zap.Error(err))
			return
		}

		userID, err := utils.GetCtxUserID(r.Context())
		if err != nil {
			sendUnauthorized(w, logger)
			return
		}

		indexOrder := models.IndexOrder{UserID: userID}

		count, err := fetchOrderCount(r.Context(), gophermart, indexOrder, logger)
		if err != nil {
			sendInternalError(w, logger)
			return
		}

		if count < 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		ordersChan, errChan := gophermart.IndexOrder(r.Context(), indexOrder)

		if !isStreamingSupported(w, logger) {
			sendInternalError(w, logger)
			return
		}

		if err := writeJSONArrayStart(w); err != nil {
			logger.Error("error writing start of json array", zap.Error(err))
			return
		}

		if err := streamOrders(w, ordersChan, errChan, r.Context(), logger); err != nil {
			logger.Error("streaming failed", zap.Error(err))
		}
	}
}

func prepareResponseWriter(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	return nil
}

func sendUnauthorized(w http.ResponseWriter, logger *zap.Logger) {
	w.WriteHeader(http.StatusUnauthorized)
	logger.Info("unauthorized access")
}

func sendInternalError(w http.ResponseWriter, logger *zap.Logger) {
	w.WriteHeader(http.StatusInternalServerError)
	logger.Error("internal server error")
}

func fetchOrderCount(
	ctx context.Context,
	gophermart Gophermart,
	indexOrder models.IndexOrder,
	logger *zap.Logger,
) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultCtxTimeout)
	defer cancel()

	count, err := gophermart.CountOrder(ctx, indexOrder)
	if err != nil {
		logger.Error("count order", zap.Error(err))
		return 0, err
	}
	return count, nil
}

func isStreamingSupported(w http.ResponseWriter, logger *zap.Logger) bool {
	_, ok := w.(http.Flusher)
	if !ok {
		logger.Error("streaming not supported")
		return false
	}
	return true
}

func writeJSONArrayStart(w http.ResponseWriter) error {
	_, err := w.Write([]byte("[\n"))
	return err
}

func streamOrders(
	w http.ResponseWriter,
	ordersChan <-chan models.IndexOrderResponse,
	errChan <-chan error,
	ctx context.Context,
	logger *zap.Logger,
) error {
	flusher := w.(http.Flusher)
	encoder := json.NewEncoder(w)
	isFirst := true

	for {
		select {
		case order, ok := <-ordersChan:
			if !ok {
				return writeJSONArrayEnd(w)
			}

			if !isFirst {
				if _, err := w.Write([]byte(",\n")); err != nil {
					return err
				}
			}
			isFirst = false

			if err := encoder.Encode(order); err != nil {
				return err
			}

			flusher.Flush()

		case err := <-errChan:
			logger.Error("stream error", zap.Error(err))
			return writeJSONArrayEnd(w)

		case <-ctx.Done():
			logger.Info("request context canceled")
			return writeJSONArrayEnd(w)
		}
	}
}

func writeJSONArrayEnd(w http.ResponseWriter) error {
	_, err := w.Write([]byte("\n]"))
	return err
}

func Store(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получение userID из контекста
		userID, err := utils.GetCtxUserID(r.Context())
		if err != nil {
			writeUnauthorized(w, logger)
			return
		}

		// 2. Декодирование и валидация запроса
		storeOrder, problems, err := decodeAndValidateRequest(r, logger)
		if err != nil {
			writeBadRequest(w, logger, err)
			return
		}
		if len(problems) > 0 {
			writeValidationError(w, logger, problems)
			return
		}

		// 3. Установка userID и сохранение заказа
		storeOrder.UserID = userID
		err = gophermart.StoreOrder(r.Context(), *storeOrder)

		// 4. Обработка ошибок сохранения
		if err != nil && !errors.Is(err, myerrors.ErrConflictNumber) {
			writeInternalError(w, logger, "error storing order", err)
			return
		}

		// 5. Определение финального статуса ответа
		status, err := determineResponseStatus(r.Context(), gophermart, storeOrder, err)
		if err != nil {
			writeInternalError(w, logger, "error determining response status", err)
			return
		}

		w.WriteHeader(status)
	}
}

func decodeAndValidateRequest(r *http.Request, logger *zap.Logger) (*models.StoreOrder, map[string]map[string]string, error) {
	storeOrderWrapper := &validators.StoreOrderWrapper{
		StoreOrder: &models.StoreOrder{},
	}

	problems, err := validators.Decode(r, storeOrderWrapper)
	if err != nil {
		logger.Debug("bad request", zap.Error(err))
		return nil, nil, fmt.Errorf("decode request: %w", err)
	}

	return storeOrderWrapper.StoreOrder, problems, nil
}

func writeUnauthorized(w http.ResponseWriter, logger *zap.Logger) {
	logger.Debug("unauthorized request")
	w.WriteHeader(http.StatusUnauthorized)
}

func writeBadRequest(w http.ResponseWriter, logger *zap.Logger, err error) {
	logger.Debug("bad request", zap.Error(err))
	w.WriteHeader(http.StatusBadRequest)
}

func writeValidationError(w http.ResponseWriter, logger *zap.Logger, problems map[string]map[string]string) {
	status := http.StatusBadRequest

	if numberProblems, ok := problems["number"]; ok {
		if _, hasNoValidError := numberProblems["is_valid"]; hasNoValidError {
			status = http.StatusUnprocessableEntity
		}
	}

	problemsStr := validators.ValidErrToStr(problems)
	logger.Debug("validation failed", zap.String("problems", problemsStr))

	w.WriteHeader(status)
	_, err := w.Write([]byte(problemsStr))
	if err != nil {
		logger.Error("failed to write validation error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func writeInternalError(w http.ResponseWriter, logger *zap.Logger, msg string, err error) {
	logger.Error(msg, zap.Error(err))
	w.WriteHeader(http.StatusInternalServerError)
}

func determineResponseStatus(
	ctx context.Context,
	gophermart Gophermart,
	storeOrder *models.StoreOrder,
	err error,
) (int, error) {
	if errors.Is(err, myerrors.ErrConflictNumber) {
		order, err := gophermart.GetOrder(ctx, models.GetOrder{
			Number: storeOrder.Number,
		})
		if err != nil {
			return 0, fmt.Errorf("getting order %s: %w", storeOrder.Number, err)
		}

		if order == nil {
			return 0, myerrors.ErrOrderNotFound
		}

		if order.UserID == storeOrder.UserID {
			return http.StatusOK, nil
		}
		return http.StatusConflict, nil
	}

	return http.StatusAccepted, nil
}
