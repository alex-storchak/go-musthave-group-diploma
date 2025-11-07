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

type Accrual struct {
	cfg        *config.Config
	retryUntil time.Time
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

func newRetryAfter(n time.Duration) error {
	return fmt.Errorf("%w %d", myerrors.ErrAccrualRetryAfter, n)
}

func newErrorRequest(code int) error {
	return fmt.Errorf("%w status %d", myerrors.ErrAccrual, code)
}

// GetRetryAfter возвращает оставшееся время паузы.
func (c *Accrual) GetRetryAfter() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.retryUntil.IsZero() {
		return 0
	}
	now := time.Now()
	if now.After(c.retryUntil) {
		return 0
	}
	return c.retryUntil.Sub(now)
}

// SetRetryAfter устанавливает паузу
func (c *Accrual) SetRetryAfter(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d <= 0 {
		c.retryUntil = time.Time{}
		return
	}
	newUntil := time.Now().Add(d)
	if newUntil.Before(c.retryUntil) {
		return
	}
	c.retryUntil = newUntil
}

//nolint:cyclop // clear enough
func (c *Accrual) Get(ctx context.Context, order models.OrderProcess) (*models.AccrualResponse, error) {

	if d := c.GetRetryAfter(); d > 0 {
		return nil, newRetryAfter(d)
	}

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
		c.SetRetryAfter(d)
		return nil, newRetryAfter(d)
	case http.StatusNoContent:
		return nil, myerrors.ErrAccrualNoOrder

	default:
		return nil, newErrorRequest(resp.StatusCode)
	}
}
