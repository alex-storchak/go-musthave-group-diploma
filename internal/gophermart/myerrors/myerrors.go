package myerrors

import "errors"

var (
	ErrOrderNotFound  = errors.New("not found order")
	ErrConflictNumber = errors.New("number conflict")
)
