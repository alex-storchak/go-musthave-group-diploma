package withdrawals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Gophermart interface {
	StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal, setDefaultBalance models.SetDefaultBalanceRequest) error
	IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (<-chan models.IndexWithdrawalResponse, <-chan error)
	CountWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (int64, error)
}

func Index(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := int64(1)
		indexWithdrawal := models.IndexWithdrawal{UserID: userID}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

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

		withdrawalChan, errChan := gophermart.IndexWithdrawal(ctx, indexWithdrawal)

		flusher, ok := w.(http.Flusher)
		if !ok {
			logger.Error("streaming not supported")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte("[\n"))
		if err != nil {
			logger.Error("writing start of json array", zap.Error(err))
			return
		}

		encoder := json.NewEncoder(w)
		isFirst := true

		for {
			select {
			case withdrawal, ok := <-withdrawalChan:
				if !ok {
					if _, err = w.Write([]byte("\n]")); err != nil {
						logger.Error("writing end of json array", zap.Error(err))
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

				if err = encoder.Encode(withdrawal); err != nil {
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

		storeWithdrawalWrapper := &validators.StoreWithdrawalWrapper{
			StoreWithdrawalRequest: &models.StoreWithdrawalRequest{},
		}

		problems, err := validators.Decode(r, storeWithdrawalWrapper)
		if err != nil {
			logger.Debug("bad request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(problems) > 0 {
			logger.Debug("bad request", zap.String("problems", fmt.Sprintf("%v", problems)))
			myerrors.ErrorValidateJSONResponse(w, problems, http.StatusBadRequest)
			return
		}

		userID := int64(1)

		err = gophermart.StoreWithdrawal(r.Context(), models.StoreWithdrawal{
			Number: storeWithdrawalWrapper.StoreWithdrawalRequest.Number,
			Sum:    storeWithdrawalWrapper.StoreWithdrawalRequest.Sum,
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
