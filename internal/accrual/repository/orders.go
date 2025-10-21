package repository

import (
	"context"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
)

func NewPgOrderRepository(db config.DB) PgOrderRepository {
	// TODO: implement me
	panic("implement me")
	return PgOrderRepository{}
}

type PgOrderRepository struct {
}

func (p PgOrderRepository) Add(ctx context.Context, order model.Order) error {
	// TODO implement me
	panic("implement me")
}

func (p PgOrderRepository) Has(ctx context.Context, number string) (bool, error) {
	// TODO implement me
	panic("implement me")
}
