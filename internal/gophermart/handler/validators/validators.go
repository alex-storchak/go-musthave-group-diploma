package validators

import (
	"context"
	"strconv"
	"strings"
)

type Validator interface {
	Valid(ctx context.Context) map[string]string
}

// Функция проверки номера по алгоритму Луна
func isValidLuhn(number string) bool {
	// Удаляем все нецифровые символы
	number = strings.ReplaceAll(number, " ", "")
	number = strings.ReplaceAll(number, "-", "")

	sum := 0
	isEven := false

	// Проходим по номеру справа налево
	for i := len(number) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(number[i]))

		if isEven {
			digit *= 2
			if digit > 9 {
				digit = digit - 9
			}
		}

		sum += digit
		isEven = !isEven
	}

	return sum%10 == 0
}
