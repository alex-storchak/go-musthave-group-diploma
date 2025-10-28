package withdrawals

import (
	"context"
	"errors"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"go.uber.org/zap"
	"net/http"
)

type Gophermart interface {
	StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal) error
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

		err = gophermart.StoreWithdrawal(r.Context(), models.StoreWithdrawal{
			Number: storeWithdrawalWrapper.StoreWithdrawalRequest.Number,
			Sum:    storeWithdrawalWrapper.StoreWithdrawalRequest.Sum,
			UserID: 1,
		})
		isErrConflictNumber := errors.Is(err, myerrors.ErrConflictNumber)
		if err != nil && !isErrConflictNumber {
			logger.Error("error storing order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
