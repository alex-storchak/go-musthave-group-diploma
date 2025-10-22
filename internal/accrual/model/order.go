package model

import (
	"database/sql"
	"time"
)

type Good struct {
	Description string `db:"description"`
	Price       int    `db:"price"`
}

type Order struct {
	Number       string       `json:"order" db:"order_number"`
	Status       string       `json:"status" db:"status"`
	Accrual      float64      `json:"accrual,omitempty" db:"accrual"`
	Goods        []Good       `json:"-"`
	RegisteredAt time.Time    `json:"-" db:"registered_at"`
	ProcessedAt  sql.NullTime `json:"-" db:"processed_at"`
}
