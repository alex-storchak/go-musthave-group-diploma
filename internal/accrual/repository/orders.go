package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPgOrders(pool *pgxpool.Pool, l *zap.Logger) *PgOrders {
	return &PgOrders{
		dbPool: pool,
		logger: l,
	}
}

type PgOrders struct {
	dbPool *pgxpool.Pool
	logger *zap.Logger
}

func (p *PgOrders) Add(ctx context.Context, order *model.Order) error {
	qOrders := `
		INSERT INTO accrual_orders (order_number, status_id)
		VALUES ($1, (SELECT id FROM public.accrual_statuses aos WHERE aos.code = 'REGISTERED')) RETURNING id
	`
	qOrderGoods := `INSERT INTO order_goods (order_id, description, price) VALUES ($1, $2, $3)`

	trx, err := p.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("error on begin transaction: %w", err)
	}
	defer func() {
		if rErr := trx.Rollback(ctx); rErr != nil {
			if !errors.Is(rErr, pgx.ErrTxClosed) {
				p.logger.Error("rollback transaction", zap.Error(rErr))
			}
		}
	}()

	var orderID int64
	err = trx.QueryRow(ctx, qOrders, order.Number).Scan(&orderID)
	if err != nil {
		return fmt.Errorf("insert order (%s) and scan returning order id: %w", order.Number, err)
	}

	batch := pgx.Batch{}
	for _, g := range order.Goods {
		batch.Queue(qOrderGoods, orderID, g.Description, g.Price)
	}
	br := trx.SendBatch(ctx, &batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("close goods batch: %w", err)
	}
	if err := trx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (p *PgOrders) Has(ctx context.Context, number string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM accrual_orders ao WHERE ao.order_number = $1)`

	var has bool
	row := p.dbPool.QueryRow(ctx, q, number)
	err := row.Scan(&has)
	if err != nil {
		return false, fmt.Errorf("scan query result row: %w", err)
	}
	return has, nil
}

func (p *PgOrders) Close() {
	p.dbPool.Close()
}
