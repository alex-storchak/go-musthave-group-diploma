package validators

import (
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/codec"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"net/http"
)

type StoreWithdrawalWrapper struct {
	*models.StoreWithdrawalRequest
}

func (o *StoreWithdrawalWrapper) Valid(r *http.Request) (map[string]map[string]string, error) {
	problems := make(map[string]map[string]string)

	// 1. Декодирование JSON
	err := decodeRequest(r, &o.StoreWithdrawalRequest)
	if err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	// 2. Валидация номера
	numberProblems := validateNumberWithdrawal(o.Number)
	if len(numberProblems) > 0 {
		problems["number"] = numberProblems
		o.Number = ""
	}

	// 3. Валидация суммы
	sumProblems := validateSum(float64(o.Sum))
	if len(sumProblems) > 0 {
		problems["sum"] = sumProblems
		o.Sum = 0
	}

	return problems, nil
}

func decodeRequest(r *http.Request, target **models.StoreWithdrawalRequest) error {
	defer func() {
		_ = r.Body.Close() // Гарантированное закрытие тела запроса
	}()

	req, err := codec.Decode[*models.StoreWithdrawalRequest](r)
	if err != nil {
		return err
	}
	*target = req
	return nil
}

func validateNumberWithdrawal(number string) map[string]string {
	problems := make(map[string]string)

	if number == "" {
		problems["required"] = "number is required"
		return problems
	}

	if !validator.IsNumericRegex(number) {
		problems["is_numeric"] = "number must contain only digits"
	}

	if !validator.IsValidLuhn(number) {
		problems["is_valid"] = "the number is not valid according to the Luhn algorithm"
	}

	return problems
}

func validateSum(sum float64) map[string]string {
	problems := make(map[string]string)

	if sum == 0 {
		problems["required"] = "no sum"
	}

	return problems
}
