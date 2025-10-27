package repository

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
)

type Repository interface {
	Ping(ctx context.Context) error
	SetOrder(ctx context.Context, storeOrder models.StoreOrder) error
	GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error)
	IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.Order, <-chan error)
}
