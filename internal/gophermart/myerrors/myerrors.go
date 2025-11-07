package myerrors

import (
	"encoding/json"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"net/http"
)

var (
	ErrOrderNotFound      = errors.New("not found order")
	ErrBalanceNotFound    = errors.New("not found balance")
	ErrConflictNumber     = errors.New("number conflict")
	ErrBalance            = errors.New("insufficient funds")
	ErrAccrualNoOrder     = errors.New("no order")
	ErrAccrual            = errors.New("error accrual")
	ErrAccrualResponseNil = errors.New("accrual response is nil")
	ErrOrderNumberNil     = errors.New("order number is empty")
	ErrValidation         = errors.New("validation error")
)

func ErrorValidateJSONResponse(w http.ResponseWriter, messages map[string]map[string]string, code int) {
	errResp := models.ErrorJSONResponse{
		Message: messages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(errResp)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
