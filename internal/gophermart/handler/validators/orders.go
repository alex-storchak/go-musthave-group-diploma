package validators

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type StoreOrder struct {
	UserID int64  `json:"user_id"`
	Number string `json:"order_number"`
}

func (r *StoreOrder) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)

	if r.Number == "" {
		problems["order_number_required"] = "where is number"
	}

	if !isValidLuhn(r.Number) {
		problems["order_number_valid"] = "the number is not valid"
	}

	return problems
}

func DecodeStoreOrder(r *http.Request) (StoreOrder, map[string]string, error) {
	var order StoreOrder

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return order, nil, fmt.Errorf("error reading request body: %w", err)
	}
	defer r.Body.Close()

	order.Number = string(body)

	if problems := order.Valid(r.Context()); len(problems) > 0 {
		return order, problems, fmt.Errorf("invalid %T: %d problems", order, len(problems))
	}
	return order, nil, nil
}
