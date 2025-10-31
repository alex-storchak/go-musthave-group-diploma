package gophermart

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg"
	"gorm.io/gorm"
)

type Gophermart struct {
	store repository.Repository
}

func NewGophermart(conn *gorm.DB) (*Gophermart, error) {
	store, err := NewRepository(conn)
	if err != nil {
		return nil, fmt.Errorf("no init repository: %w", err)
	}

	return &Gophermart{
		store: store,
	}, nil
}

func NewRepository(conn *gorm.DB) (repository.Repository, error) {
	return pg.NewStore(conn)
}

func (f *Gophermart) Ping(ctx context.Context) error {
	return f.store.Ping(ctx)
}

func (f *Gophermart) CountOrder(ctx context.Context, indexOrder models.IndexOrder) (int64, error) {
	return f.store.CountOrder(ctx, indexOrder)
}

func (f *Gophermart) IndexOrder(ctx context.Context, indexOrder models.IndexOrder) ([]models.IndexOrderResponse, error) {
	return f.store.IndexOrder(ctx, indexOrder)
}

func (f *Gophermart) StoreOrder(ctx context.Context, storeOrder models.StoreOrder) error {
	return f.store.SetOrder(ctx, storeOrder)
}

func (f *Gophermart) GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error) {
	return f.store.GetOrder(ctx, getOrder)
}

func (f *Gophermart) GetBalance(ctx context.Context, getBalance models.GetBalanceRequest) (*models.ShowBalanceResponse, error) {
	return f.store.GetBalance(ctx, getBalance)
}

func (f *Gophermart) CreateDefaultBalance(ctx context.Context, setDefaultBalance models.SetDefaultBalanceRequest) error {
	return f.store.SetDefaultBalance(ctx, setDefaultBalance)
}

func (f *Gophermart) StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal, setDefaultBalance models.SetDefaultBalanceRequest) error {
	return f.store.StoreWithdrawal(ctx, storeWithdrawal, setDefaultBalance)
}

func (f *Gophermart) CountWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (int64, error) {
	return f.store.CountWithdrawal(ctx, indexWithdrawal)
}

func (f *Gophermart) IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) ([]models.IndexWithdrawalResponse, error) {
	return f.store.IndexWithdrawal(ctx, indexWithdrawal)
}
