package ping

import (
	"context"
	"go.uber.org/zap"
	"net/http"
)

type Gophermart interface {
	Ping(ctx context.Context) error
}

func Ping(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := gophermart.Ping(r.Context())
		if err != nil {
			logger.Error("ping store error", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
