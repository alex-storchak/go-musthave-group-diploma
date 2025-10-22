package validator

import (
	"strconv"
	"unicode"
)

//nolint:mnd // numbers just the need for the algorithm
func IsValidLuhn(number string) bool {
	cleanNumber := ""
	for _, char := range number {
		if unicode.IsDigit(char) {
			cleanNumber += string(char)
		} else if !unicode.IsSpace(char) {
			return false
		}
	}

	if len(cleanNumber) < 2 {
		return false
	}

	sum := 0
	isSecond := false

	for i := len(cleanNumber) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(cleanNumber[i]))
		if err != nil {
			return false
		}

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}
