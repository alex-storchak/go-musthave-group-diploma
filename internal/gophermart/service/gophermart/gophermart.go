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

func (f *Gophermart) IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.Order, <-chan error) {
	return f.store.IndexOrder(ctx, indexOrder)
}

func (f *Gophermart) StoreOrder(ctx context.Context, storeOrder models.StoreOrder) error {
	return f.store.SetOrder(ctx, storeOrder)
}

func (f *Gophermart) GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error) {
	return f.store.GetOrder(ctx, getOrder)
}
