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
		return nil, fmt.Errorf("init repository: %w", err)
	}

	return &Gophermart{
		store: store,
	}, nil
}

func NewRepository(conn *gorm.DB) (repository.Repository, error) {
	repo, err := pg.NewStore(conn)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}
	return repo, nil
}

func (f *Gophermart) Ping(ctx context.Context) error {
	err := f.store.Ping(ctx)
	if err != nil {
		return fmt.Errorf("ping repository: %w", err)
	}
	return nil
}

func (f *Gophermart) CountOrder(ctx context.Context, indexOrder models.IndexOrder) (int64, error) {
	count, err := f.store.CountOrder(ctx, indexOrder)
	if err != nil {
		return 0, fmt.Errorf("count orders: %w", err)
	}
	return count, nil
}

func (f *Gophermart) IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.IndexOrderResponse, <-chan error) {
	return f.store.IndexOrder(ctx, indexOrder)
}

func (f *Gophermart) StoreOrder(ctx context.Context, storeOrder models.StoreOrder) error {
	err := f.store.SetOrder(ctx, storeOrder)
	if err != nil {
		return fmt.Errorf("store order: %w", err)
	}
	return nil
}

func (f *Gophermart) GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error) {
	order, err := f.store.GetOrder(ctx, getOrder)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return order, nil
}

func (f *Gophermart) GetBalance(ctx context.Context, getBalance models.GetBalanceRequest) (*models.ShowBalanceResponse, error) {
	balance, err := f.store.GetBalance(ctx, getBalance)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

func (f *Gophermart) CreateDefaultBalance(ctx context.Context, setDefaultBalance models.SetDefaultBalanceRequest) error {
	err := f.store.SetDefaultBalance(ctx, setDefaultBalance)
	if err != nil {
		return fmt.Errorf("create default balance: %w", err)
	}
	return nil
}

func (f *Gophermart) StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal, setDefaultBalance models.SetDefaultBalanceRequest) error {
	err := f.store.StoreWithdrawal(ctx, storeWithdrawal, setDefaultBalance)
	if err != nil {
		return fmt.Errorf("store withdrawal: %w", err)
	}
	return nil
}

func (f *Gophermart) CountWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (int64, error) {
	count, err := f.store.CountWithdrawal(ctx, indexWithdrawal)
	if err != nil {
		return 0, fmt.Errorf("count withdrawal: %w", err)
	}
	return count, nil
}

func (f *Gophermart) IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) ([]models.IndexWithdrawalResponse, error) {
	withdrawals, err := f.store.IndexWithdrawal(ctx, indexWithdrawal)
	if err != nil {
		return nil, fmt.Errorf("index withdrawal: %w", err)
	}
	return withdrawals, nil
}
