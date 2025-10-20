package pg

import (
	"context"
	"database/sql"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg/migrator"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Store struct {
	cfg  *config.Config
	conn *sql.DB
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	conn, err := sql.Open("pgx", cfg.DatabaseDsn)
	if err != nil {
		return nil, err
	}

	err = migrator.ApplyMigrations(conn, "file://./migrations/gophermart")
	if err != nil {
		return nil, err
	}

	return &Store{
		cfg:  cfg,
		conn: conn,
	}, nil
}

func (st *Store) Ping(ctx context.Context) error {
	return st.conn.PingContext(ctx)
}

func (st *Store) Close() error {
	return st.conn.Close()
}
