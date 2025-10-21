package validator

import (
	"fmt"
)

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any
	// problems. If len(problems) == 0 then
	// the object is valid.
	Valid() (problems Problems)
}

type Problems map[string]string

func IsValid[T Validator](v T) (problems Problems, err error) {
	problems = v.Valid()
	if len(problems) > 0 {
		err = fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return
}
