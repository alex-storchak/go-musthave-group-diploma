package models

import "time"

type StoreWithdrawalRequest struct {
	Number string  `json:"order" gorm:"column:order_number;primaryKey"`
	Sum    float64 `json:"sum"`
}

type StoreWithdrawal struct {
	Number string  `json:"order_number" gorm:"column:order_number;primaryKey"`
	Sum    float64 `json:"sum"`
	UserID UserID  `json:"user_id"`
}

type IndexWithdrawal struct {
	UserID UserID `json:"user_id"`
}

type IndexWithdrawalResponse struct {
	Number      string     `json:"order" gorm:"column:order_number;primaryKey"`
	Sum         float64    `json:"sum"`
	ProcessedAt *time.Time `json:"processed_at"`
}
