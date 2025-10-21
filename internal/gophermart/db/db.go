package db

import (
	"database/sql"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/config"
)

func InitDB(c *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", c.DatabaseDsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
