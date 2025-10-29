package myerrors

import (
	"encoding/json"
	"errors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"net/http"
)

var (
	ErrOrderNotFound   = errors.New("not found order")
	ErrBalanceNotFound = errors.New("not found balance")
	ErrConflictNumber  = errors.New("number conflict")
	ErrBalance         = errors.New("insufficient funds")
)

func ErrorValidateJSONResponse(w http.ResponseWriter, messages map[string]map[string]string, code int) {
	errResp := models.ErrorJSONResponse{
		Message: messages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errResp)
}
