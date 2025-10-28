package repository

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
)

type Repository interface {
	Ping(ctx context.Context) error
	SetOrder(ctx context.Context, storeOrder models.StoreOrder) error
	GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error)
	GetOrderUser(ctx context.Context, getOrderUser models.GetOrderUser) (*models.Order, error)
	IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.IndexOrderResponse, <-chan error)

	GetBalance(ctx context.Context, getBalance models.GetBalanceRequest) (*models.ShowBalanceResponse, error)
	SetDefaultBalance(ctx context.Context, setDefaultBalance models.SetDefaultBalanceRequest) error
}
