package validators

import (
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"io"
	"net/http"
)

type StoreOrderWrapper struct {
	*models.StoreOrder
}

func (o *StoreOrderWrapper) Valid(r *http.Request) (map[string]map[string]string, error) {
	problems := make(map[string]map[string]string)

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		return problems, fmt.Errorf("error reading request body: %w", err)
	}
	defer r.Body.Close()

	number := utils.RemoveWhitespaces(string(body))

	numberProblem := make(map[string]string)
	if number == "" {
		numberProblem["required"] = "number is required"
	} else if !utils.IsNumericRegex(number) {
		numberProblem["is_numeric"] = "number must contain only digits"
	} else if !utils.IsValidLuhn(number) {
		numberProblem["is_valid"] = "the number is not valid according to the Luhn algorithm"
	}

	if len(numberProblem) > 0 {
		problems["number"] = numberProblem
	}

	if len(problems["number"]) == 0 {
		o.StoreOrder.Number = number
	}

	return problems, nil
}
