package models

import "github.com/alex-storchak/go-musthave-group-diploma/internal/validator"

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r AuthRequest) Valid() validator.Problems {
	problems := make(validator.Problems)

	if r.Login == "" {
		problems["login"] = "login is required"
	}

	if r.Password == "" {
		problems["password"] = "password is required"
	}

	return problems
}

type AuthResponse struct {
	Message string `json:"message"`
	UserID  uint   `json:"user_id,omitempty"`
	Login   string `json:"login,omitempty"`
}
