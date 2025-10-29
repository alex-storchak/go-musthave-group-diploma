package models

import "time"

type OrderStatus string

const (
	OrderNew        OrderStatus = "NEW"
	OrderProcessing             = "PROCESSING"
	OrderInvalid                = "INVALID"
	OrderProcessed              = "PROCESSED"
)

type Order struct {
	Number      string      `json:"order_number" gorm:"column:order_number;primaryKey"`
	UserID      UserID      `json:"user_id"`
	Status      OrderStatus `json:"status"`
	Accrual     float64     `json:"accrual,omitempty"`
	UploadedAt  *time.Time  `json:"uploaded_at" gorm:"default:current_timestamp"`
	ProcessedAt *time.Time  `json:"processed_at,omitempty"`
}

type IndexOrderResponse struct {
	Number     string     `json:"number" gorm:"column:order_number;primaryKey"`
	Status     string     `json:"status"`
	Accrual    float64    `json:"accrual,omitempty"`
	UploadedAt *time.Time `json:"uploaded_at"`
}

type IndexOrder struct {
	UserID UserID `json:"user_id"`
}

type StoreOrder struct {
	UserID UserID `json:"user_id"`
	Number string `json:"order_number"`
}

type GetOrder struct {
	Number string `json:"order_number"`
}

type GetOrderUser struct {
	Number string `json:"order_number"`
	UserID UserID `json:"user_id"`
}
