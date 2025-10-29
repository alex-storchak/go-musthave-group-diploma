package validators

import (
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/utils"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"net/http"
)

type StoreWithdrawalWrapper struct {
	*models.StoreWithdrawalRequest
}

func (o *StoreWithdrawalWrapper) Valid(r *http.Request) (map[string]map[string]string, error) {
	problems := make(map[string]map[string]string)

	var err error
	o.StoreWithdrawalRequest, err = codec.Decode[*models.StoreWithdrawalRequest](r)
	if err != nil {
		return problems, fmt.Errorf("decode json: %w", err)
	}

	numberProblem := make(map[string]string)
	if o.StoreWithdrawalRequest.Number == "" {
		numberProblem["required"] = "number is required"
	} else if !utils.IsNumericRegex(o.StoreWithdrawalRequest.Number) {
		numberProblem["is_numeric"] = "number must contain only digits"
	} else if !utils.IsValidLuhn(o.StoreWithdrawalRequest.Number) {
		numberProblem["is_valid"] = "the number is not valid according to the Luhn algorithm"
	}

	if len(numberProblem) > 0 {
		problems["number"] = numberProblem
		o.StoreWithdrawalRequest.Number = ""
	}

	sumProblem := make(map[string]string)
	if o.StoreWithdrawalRequest.Sum == 0 {
		sumProblem["required"] = "no sum"
	}

	if len(sumProblem) > 0 {
		problems["sum"] = sumProblem
		o.StoreWithdrawalRequest.Sum = 0
	}

	return problems, nil
}
