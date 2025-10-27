package utils

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func RemoveWhitespaces(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func IsNumericRegex(s string) bool {
	re := regexp.MustCompile(`^\d+$`)
	return re.MatchString(s)
}

// IsValidLuhn Функция проверки номера по алгоритму Луна
func IsValidLuhn(number string) bool {
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
