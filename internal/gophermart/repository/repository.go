package repository

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
)

type Repository interface {
	Ping(ctx context.Context) error
	SetOrder(ctx context.Context, storeOrder models.StoreOrder) error
	UpdateOrderInvalid(ctx context.Context, accrualResponse *models.AccrualResponse) error
	UpdateOrderProcessed(ctx context.Context, accrualResponse *models.AccrualResponse) error
	GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error)
	GetOrderUser(ctx context.Context, getOrderUser models.GetOrderUser) (*models.Order, error)
	GetNewOrders(ctx context.Context, orders []models.OrderProcess) ([]models.OrderProcess, error)
	IndexOrder(ctx context.Context, indexOrder models.IndexOrder) ([]models.IndexOrderResponse, error)
	CountOrder(ctx context.Context, indexOrder models.IndexOrder) (int64, error)

	GetBalance(ctx context.Context, getBalance models.GetBalanceRequest) (*models.ShowBalanceResponse, error)
	SetDefaultBalance(ctx context.Context, setDefaultBalance models.SetDefaultBalanceRequest) error

	StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal, setDefaultBalance models.SetDefaultBalanceRequest) error
	IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) ([]models.IndexWithdrawalResponse, error)
	CountWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (int64, error)
}
