package models

import "time"

type Order struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Number      string    `json:"order_number"`
	StatusID    int8      `json:"status_id"`
	UploadedAt  time.Time `json:"uploaded_at"`
	ProcessedAt time.Time `json:"processed_at"`
}
