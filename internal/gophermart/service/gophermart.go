package service

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg"
)

type Gophermart struct {
	store repository.Repository
}

func NewGophermart(cfg *config.Config) (*Gophermart, error) {
	store, err := NewRepository(cfg)
	if err != nil {
		return nil, err
	}

	return &Gophermart{
		store: store,
	}, nil
}

func NewRepository(cfg *config.Config) (repository.Repository, error) {
	return pg.NewStore(cfg)
}

func (f *Gophermart) Close() error {
	return f.store.Close()
}

func (f *Gophermart) Ping(ctx context.Context) error {
	return f.store.Ping(ctx)
}
