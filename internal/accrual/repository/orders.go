package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	StatusRegistered = "REGISTERED"
	StatusProcessing = "PROCESSING"
	StatusProcessed  = "PROCESSED"
	StatusInvalid    = "INVALID"
)

var ErrOrderNotFound = errors.New("order not found")

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
		VALUES ($1, (SELECT id FROM accrual_statuses aos WHERE aos.code = $2)) RETURNING id
	`
	qOrderGoods := `INSERT INTO order_goods (order_id, description, price) VALUES ($1, $2, $3)`

	tx, err := p.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction for add order: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil {
			if !errors.Is(rErr, pgx.ErrTxClosed) {
				p.logger.Error("error on rollback transaction for add order", zap.Error(rErr))
			}
		}
	}()

	var orderID int64
	err = tx.QueryRow(ctx, qOrders, order.Number, StatusRegistered).Scan(&orderID)
	if err != nil {
		return fmt.Errorf("insert order (%s) and scan returning order id: %w", order.Number, err)
	}

	batch := pgx.Batch{}
	for _, g := range order.Goods {
		batch.Queue(qOrderGoods, orderID, g.Description, g.Price)
	}
	br := tx.SendBatch(ctx, &batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("close goods batch: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction for add order: %w", err)
	}
	return nil
}

func (p *PgOrders) Has(ctx context.Context, number string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM accrual_orders ao WHERE ao.order_number = $1)`

	var has bool
	row := p.dbPool.QueryRow(ctx, q, number)
	err := row.Scan(&has)
	if err != nil {
		return false, fmt.Errorf("scan order exists query result row: %w", err)
	}
	return has, nil
}

func (p *PgOrders) Get(ctx context.Context, number string) (*model.Order, error) {
	q := `
		SELECT 
		    ao.order_number, 
		    aos.code as status, 
		    coalesce(ao.accrual, 0) as accrual, 
		    ao.registered_at, 
		    ao.processed_at 
		FROM accrual_orders ao
		JOIN accrual_statuses aos ON ao.status_id = aos.id
		WHERE ao.order_number = $1
	`

	var order model.Order
	row := p.dbPool.QueryRow(ctx, q, number)
	err := row.Scan(&order.Number, &order.Status, &order.Accrual, &order.RegisteredAt, &order.ProcessedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	} else if err != nil {
		return nil, fmt.Errorf("scan order select query result row: %w", err)
	}
	return &order, nil
}

func (p *PgOrders) GetBatchForProcessing(ctx context.Context, batchSize int) ([]model.Order, error) {
	q := `
		WITH selected_orders AS (
			SELECT id 
			FROM accrual_orders 
			WHERE status_id = (SELECT id FROM accrual_statuses WHERE code = $2)
			ORDER BY registered_at ASC 
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE accrual_orders 
		SET status_id = (SELECT id FROM accrual_statuses WHERE code = $3)
		WHERE id IN (SELECT id FROM selected_orders)
		RETURNING 
			accrual_orders.order_number,
			coalesce(accrual_orders.accrual, 0) accrual,
			accrual_orders.registered_at,
			accrual_orders.processed_at,
			(
				SELECT COALESCE(JSON_AGG(
					JSON_BUILD_OBJECT(
						'description', og.description,
						'price', og.price
					)
				)::jsonb, '[]'::jsonb)
				FROM order_goods og 
				WHERE og.order_id = accrual_orders.id
			) as goods
	`

	tx, err := p.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction for get batch with goods: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil {
			if !errors.Is(rErr, pgx.ErrTxClosed) {
				p.logger.Error("error on rollback transaction for get batch with goods", zap.Error(rErr))
			}
		}
	}()

	rows, err := tx.Query(ctx, q, batchSize, StatusRegistered, StatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("query order batch with goods for update: %w", err)
	}
	defer rows.Close()

	orders := make([]model.Order, 0, batchSize)
	for rows.Next() {
		var order model.Order
		sErr := rows.Scan(&order.Number, &order.Accrual, &order.RegisteredAt, &order.ProcessedAt, &order.Goods)
		if sErr != nil {
			return nil, fmt.Errorf("scan order batch with goods: %w", sErr)
		}
		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("scan order batch from db: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit order batch with goods update: %w", err)
	}
	return orders, nil
}

func (p *PgOrders) UpdateStatus(ctx context.Context, order *model.Order) error {
	q := `
		UPDATE accrual_orders 
		SET 
		    status_id = (SELECT id FROM accrual_statuses WHERE code = $2), 
		    accrual = $3,
			processed_at = CURRENT_TIMESTAMP
		WHERE order_number = $1
	`

	_, err := p.dbPool.Exec(ctx, q, order.Number, order.Status, order.Accrual)
	if err != nil {
		return fmt.Errorf("update order status(%v): %w", order, err)
	}
	return nil
}

func (p *PgOrders) ResetStuckOrders(ctx context.Context, timeout time.Duration, batchLimit int) (int, error) {
	q := `
		WITH stuck_orders AS (
			SELECT id, order_number
			FROM accrual_orders 
			WHERE status_id = (SELECT id FROM accrual_statuses WHERE code = $2)
			AND registered_at < CURRENT_TIMESTAMP - ($3 * INTERVAL '1 second')
			ORDER BY registered_at ASC
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		)
		UPDATE accrual_orders 
		SET 
			status_id = (SELECT id FROM accrual_statuses WHERE code = $1),
			processed_at = NULL
		WHERE id IN (SELECT id FROM stuck_orders)
		RETURNING order_number
	`

	tx, err := p.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin transaction for reset stuck orders: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil {
			if !errors.Is(rErr, pgx.ErrTxClosed) {
				p.logger.Error("error on rollback transaction for reset stuck orders", zap.Error(rErr))
			}
		}
	}()

	timeoutSeconds := int64(timeout.Seconds())

	rows, err := tx.Query(ctx, q, StatusRegistered, StatusProcessing, timeoutSeconds, batchLimit)
	if err != nil {
		return 0, fmt.Errorf("query stuck orders for reset: %w", err)
	}
	defer rows.Close()

	var resetCount int
	resetOrderNumbers := make([]string, 0, batchLimit)
	for rows.Next() {
		var orderNumber string
		if err = rows.Scan(&orderNumber); err != nil {
			return 0, fmt.Errorf("scan reset order number: %w", err)
		}
		resetOrderNumbers = append(resetOrderNumbers, orderNumber)
		resetCount++
	}

	if err = rows.Err(); err != nil {
		return 0, fmt.Errorf("scan reset batch from db: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit stuck orders reset: %w", err)
	}

	if resetCount > 0 {
		p.logger.Debug("reset stuck orders to REGISTERED status",
			zap.Int("count", resetCount),
			zap.Strings("order_numbers", resetOrderNumbers),
		)
	}

	return resetCount, nil
}

func (p *PgOrders) Close() {
	p.dbPool.Close()
}
