package models

type StoreWithdrawalRequest struct {
	Number string  `json:"order" gorm:"column:order_number;primaryKey"`
	Sum    float64 `json:"sum"`
}

type StoreWithdrawal struct {
	Number string  `json:"order_number" gorm:"column:order_number;primaryKey"`
	Sum    float64 `json:"sum"`
	UserID int64   `json:"user_id"`
}
