package model

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	StatusRegistered = "REGISTERED"
	StatusProcessing = "PROCESSING"
	StatusProcessed  = "PROCESSED"
	StatusInvalid    = "INVALID"
)

var ErrScanGoods = errors.New("cannot scan Goods")

type Good struct {
	Description string  `db:"description"`
	Price       float64 `db:"price"`
}

type Goods []Good

func (g *Goods) Scan(src any) error {
	if src == nil {
		*g = []Good{}
		return nil
	}

	var bytes []byte
	switch v := src.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("%w from type %T", ErrScanGoods, src)
	}

	if len(bytes) == 0 || string(bytes) == "null" {
		*g = []Good{}
		return nil
	}

	err := json.Unmarshal(bytes, g)
	if err != nil {
		return fmt.Errorf("%w because of json.Unmarshal: %w", ErrScanGoods, err)
	}
	return nil
}

type Order struct {
	Number       string             `json:"order" db:"order_number"`
	Status       string             `json:"status" db:"status"`
	Accrual      float64            `json:"accrual,omitempty" db:"accrual"`
	Goods        Goods              `json:"-"`
	RegisteredAt pgtype.Timestamptz `json:"-" db:"registered_at"`
	ProcessedAt  pgtype.Timestamptz `json:"-" db:"processed_at"`
}
