package pg

import (
	"context"
	"database/sql"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg/migrator"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Store struct {
	conn *sql.DB
}

func NewStore(conn *sql.DB) (repository.Repository, error) {
	err := migrator.ApplyMigrations(conn, "file://./migrations/gophermart")
	if err != nil {
		return nil, err
	}

	return &Store{
		conn: conn,
	}, nil
}

func (st *Store) Ping(ctx context.Context) error {
	return st.conn.PingContext(ctx)
}
