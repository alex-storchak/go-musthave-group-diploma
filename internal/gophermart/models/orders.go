package models

import "time"

type OrderStatus string

const (
	OrderNew        OrderStatus = "NEW"
	OrderProcessing             = "PROCESSING"
	OrderInvalid                = "INVALID"
	OrderProcessed              = "PROCESSED"
)

var (
	OrderStatusMap = map[OrderStatus]int8{
		OrderNew:        1,
		OrderProcessing: 2,
		OrderInvalid:    3,
		OrderProcessed:  4,
	}

	OrderStatusByIndex = map[int8]OrderStatus{
		1: OrderNew,
		2: OrderProcessing,
		3: OrderInvalid,
		4: OrderProcessed,
	}
)

type Order struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	UserID      int64      `json:"user_id" `
	Number      string     `json:"order_number" gorm:"column:order_number;uniqueIndex:orders_order_number_key"`
	StatusID    int8       `json:"status_id"`
	UploadedAt  *time.Time `json:"uploaded_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

type IndexOrderResponse struct {
	Number     string     `json:"number" gorm:"column:order_number"`
	StatusID   string     `json:"status"`
	Accrual    int        `json:"accrual"`
	UploadedAt *time.Time `json:"uploaded_at"`
}

type IndexOrder struct {
	UserID int64 `json:"user_id"`
}

type StoreOrder struct {
	UserID int64  `json:"user_id"`
	Number string `json:"order_number"`
}

type GetOrder struct {
	Number string `json:"order_number"`
}
