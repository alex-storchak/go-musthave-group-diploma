package orders

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/validators"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type Gophermart interface {
	IndexOrder(ctx context.Context) error
	StoreOrder(ctx context.Context) error
}

func Index(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//
	}
}

func mapToStringSimple(m map[string]string) string {
	result := ""
	for key, value := range m {
		result += fmt.Sprintf("%s: %s\n", key, value)
	}
	return strings.TrimSuffix(result, "\n")
}

func Store(logger *zap.Logger, gophermart Gophermart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storeOrder, problems, err := validators.DecodeStoreOrder(r)
		if err != nil {
			logger.Debug("bad request", zap.Error(err))
			if len(problems) > 0 {
				w.Write([]byte(mapToStringSimple(problems)))
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Println(storeOrder)
	}
}
