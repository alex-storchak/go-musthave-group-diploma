package orders

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	_ "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Gophermart interface {
	IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.Order, <-chan error)
	StoreOrder(ctx context.Context, storeOrder models.StoreOrder) error
	GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error)
}

func Index(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := int64(1)
		indexOrder := models.IndexOrder{UserID: userID}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		ordersChan, errChan := gophermart.IndexOrder(ctx, indexOrder)

		flusher, ok := w.(http.Flusher)
		if !ok {
			logger.Error("streaming not supported")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err := w.Write([]byte("[\n"))
		if err != nil {
			logger.Error("error writing start of json array", zap.Error(err))
			return
		}

		encoder := json.NewEncoder(w)
		isFirst := true

		for {
			select {
			case order, ok := <-ordersChan:
				if !ok {
					if _, err = w.Write([]byte("\n]")); err != nil {
						logger.Error("error writing end of json array", zap.Error(err))
					}
					return
				}

				if !isFirst {
					if _, err = w.Write([]byte(",\n")); err != nil {
						logger.Error("error writing comma", zap.Error(err))
						return
					}
				}
				isFirst = false

				if err = encoder.Encode(order); err != nil {
					logger.Error("error encoding order", zap.Error(err))
					return
				}

				flusher.Flush()

			case err = <-errChan:
				logger.Error("stream error", zap.Error(err))
				if _, err = w.Write([]byte("\n]")); err != nil {
					logger.Error("error writing end of json array", zap.Error(err))
				}
				return

			case <-ctx.Done():
				logger.Info("request context cancelled")
				if _, err = w.Write([]byte("\n]")); err != nil {
					logger.Error("error writing end of json array", zap.Error(err))
				}
				return
			}
		}
	}
}

func Store(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		storeOrderWrapper := &validators.StoreOrderWrapper{
			StoreOrder: &models.StoreOrder{},
		}

		problems, err := validators.DecodeTextPlain(r, storeOrderWrapper)
		if err != nil {
			logger.Debug("bad request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(problems) > 0 {
			status := http.StatusBadRequest

			if numberProblems, ok := problems["number"]; ok {
				if _, hasNoValidError := numberProblems["no_valid"]; hasNoValidError {
					status = http.StatusUnprocessableEntity
				}
			}
			problemsStr := validators.ValidErrToStr(problems)
			logger.Debug("bad request", zap.String("problems", problemsStr))
			w.WriteHeader(status)
			w.Write([]byte(problemsStr))
			return
		}
		storeOrderWrapper.StoreOrder.UserID = 1
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
