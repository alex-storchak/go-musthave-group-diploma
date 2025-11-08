package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual/config"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RetryAfterError struct {
	Duration time.Duration
}

func (e RetryAfterError) Error() string {
	return fmt.Sprintf("retry after: %s", e.Duration)
}

func NewRetryAfterError(d time.Duration) error {
	return &RetryAfterError{Duration: d}
}

type Accrual struct {
	cfg        *config.Config
	mu         *sync.Mutex
	httpClient *http.Client
}

func New(cfg *config.Config) *Accrual {
	return &Accrual{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		mu: &sync.Mutex{},
	}
}

func newErrorRequest(code int) error {
	return fmt.Errorf("%w status %d", myerrors.ErrAccrual, code)
}

func (c *Accrual) Get(ctx context.Context, order models.OrderProcess) (*models.AccrualResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/orders/%s", c.cfg.Addr, order.Number), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("response body: %w", err)
		}

		var res models.AccrualResponse
		if err := json.Unmarshal(body, &res); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		res.Order = order
		return &res, nil

	case http.StatusTooManyRequests:
		d := config.DefaultRetryAfter
		if hdr := resp.Header.Get("Retry-After"); hdr != "" {
			if sec, err := strconv.Atoi(strings.TrimSpace(hdr)); err == nil && sec > 0 {
				d = time.Duration(sec) * time.Second
			}
		}
		return nil, NewRetryAfterError(d)
	case http.StatusNoContent:
		return nil, myerrors.ErrAccrualNoOrder

	default:
		return nil, newErrorRequest(resp.StatusCode)
	}
}
