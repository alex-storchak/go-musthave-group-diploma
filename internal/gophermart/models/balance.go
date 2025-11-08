package models

import "time"

type GetBalanceRequest struct {
	UserID UserID `json:"user_id"`
}

type ShowBalanceResponse struct {
	Current        RoundedFloat64 `json:"current"`
	TotalWithdrawn RoundedFloat64 `json:"withdrawn"`
}

type SetDefaultBalanceRequest struct {
	UserID         UserID         `json:"user_id"`
	Current        RoundedFloat64 `json:"current"`
	TotalWithdrawn RoundedFloat64 `json:"withdrawn"`
}

type Balance struct {
	UserID         UserID         `json:"user_id" gorm:"primaryKey"`
	Current        RoundedFloat64 `json:"current"`
	TotalWithdrawn RoundedFloat64 `json:"withdrawn"`
	UpdatedAt      *time.Time     `json:"uploaded_at" gorm:"default:current_timestamp"`
}
