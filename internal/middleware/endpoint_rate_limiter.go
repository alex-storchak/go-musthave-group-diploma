package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"go.uber.org/zap"
)

func NewEndpointRateLimiter(
	limit int,
	logger *zap.Logger,
) func(next http.Handler) http.Handler {
	return httprate.Limit(
		limit,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByEndpoint),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			body := fmt.Sprintf("No more than %d requests per minute allowed", limit)
			if _, err := w.Write([]byte(body)); err != nil {
				logger.Error("failed to write endpoint rate limiter response body", zap.Error(err))
			}
		}),
	)
}
