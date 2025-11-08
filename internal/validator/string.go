package validator

import "regexp"

func IsNumericRegex(s string) bool {
	re := regexp.MustCompile(`^\d+$`)
	return re.MatchString(s)
}
