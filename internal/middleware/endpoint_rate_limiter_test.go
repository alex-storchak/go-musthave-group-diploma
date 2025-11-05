package middleware

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testOkHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestNewEndpointRateLimiter_WithinLimit(t *testing.T) {
	t.Run("should allow requests within limit", func(t *testing.T) {
		logger := zap.NewNop()
		limit := 5
		rateLimiter := NewEndpointRateLimiter(limit, logger)
		handler := rateLimiter(testOkHandler)

		for i := 0; i < limit; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		}
	})
}

func TestNewEndpointRateLimiter_OverLimit(t *testing.T) {
	t.Run("should block requests over limit", func(t *testing.T) {
		logger := zap.NewNop()
		limit := 2
		rateLimiter := NewEndpointRateLimiter(limit, logger)
		handler := rateLimiter(testOkHandler)

		for i := 0; i < limit; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		errBody := fmt.Sprintf("No more than %d requests per minute allowed", limit)
		assert.Equal(t, errBody, rr.Body.String())
		assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
		assert.Equal(t, "60", rr.Header().Get("Retry-After"))
	})
}

func TestNewEndpointRateLimiter_ByEndpoint(t *testing.T) {
	logger := zap.NewNop()

	t.Run("should rate limit by endpoint", func(t *testing.T) {
		rateLimiter := NewEndpointRateLimiter(1, logger)
		handler := rateLimiter(testOkHandler)

		req1 := httptest.NewRequest("GET", "/test1", nil)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		assert.Equal(t, http.StatusOK, rr1.Code)

		// same endpoint -> fail
		req2 := httptest.NewRequest("GET", "/test1", nil)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		assert.Equal(t, http.StatusTooManyRequests, rr2.Code)

		// different endpoint -> success
		req3 := httptest.NewRequest("GET", "/test2", nil)
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, req3)
		assert.Equal(t, http.StatusOK, rr3.Code)
	})
}
