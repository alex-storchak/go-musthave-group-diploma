package gophermart

import (
	"context"
	"database/sql"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg"
)

type Gophermart struct {
	store repository.Repository
}

func NewGophermart(conn *sql.DB) (*Gophermart, error) {
	store, err := NewRepository(conn)
	if err != nil {
		return nil, err
	}

	return &Gophermart{
		store: store,
	}, nil
}

func NewRepository(conn *sql.DB) (repository.Repository, error) {
	return pg.NewStore(conn)
}

func (f *Gophermart) Ping(ctx context.Context) error {
	return f.store.Ping(ctx)
}

func (f *Gophermart) IndexOrder(ctx context.Context) error {
	return nil
}
func (f *Gophermart) StoreOrder(ctx context.Context) error {
	return nil
}
