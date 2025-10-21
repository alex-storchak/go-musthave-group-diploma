package model

import "time"

type Good struct {
	Description string
	Price       int
}

type Order struct {
	Number       string  `json:"order"`
	Status       string  `json:"status"`
	Accrual      float64 `json:"accrual"`
	Goods        []Good
	RegisteredAt time.Time
	ProcessedAt  time.Time
}
