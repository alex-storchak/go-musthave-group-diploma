package models

import "time"

type GetBalanceRequest struct {
	UserID int64 `json:"user_id"`
}

type ShowBalanceResponse struct {
	Current        float64 `json:"current"`
	TotalWithdrawn float64 `json:"withdrawn"`
}

type SetDefaultBalanceRequest struct {
	UserID         int64   `json:"user_id"`
	Current        float64 `json:"current"`
	TotalWithdrawn float64 `json:"withdrawn"`
}

type Balance struct {
	UserID         int64      `json:"user_id" gorm:"primaryKey"`
	Current        float64    `json:"current"`
	TotalWithdrawn float64    `json:"withdrawn"`
	UpdatedAt      *time.Time `json:"uploaded_at" gorm:"default:current_timestamp"`
}
