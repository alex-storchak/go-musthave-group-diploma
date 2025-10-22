package factory

import (
	"context"
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/service"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Repo interface {
	MakeOrders() service.OrdersRepository
}

func NewPgRepo(ctx context.Context, cfg config.DB, l *zap.Logger) (Repo, error) {
	pgCfg := db.PgConfig{
		DSN:            cfg.DatabaseURI,
		MigrationsPath: cfg.MigrationsPath,
	}
	pgDB, err := db.NewPgDB(ctx, pgCfg, l)
	if err != nil {
		return nil, fmt.Errorf("create pg db pool: %w", err)
	}
	return &PgRepo{
		dbPool: pgDB,
		config: pgCfg,
		logger: l,
	}, nil
}

type PgRepo struct {
	dbPool *pgxpool.Pool
	config db.PgConfig
	logger *zap.Logger
}

func (p *PgRepo) MakeOrders() service.OrdersRepository {
	return repository.NewPgOrders(p.dbPool, p.logger)
}
