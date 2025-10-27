package pg

import (
	"context"
	"errors"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/myerrors"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg/migrator"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/gorm"
	"time"
)

type Store struct {
	conn *gorm.DB
}

func NewStore(conn *gorm.DB) (repository.Repository, error) {
	err := migrator.ApplyMigrations(conn, "file://./migrations/gophermart")
	if err != nil {
		return nil, fmt.Errorf("no migrations: %w", err)
	}

	return &Store{
		conn: conn,
	}, nil
}

func (st *Store) Ping(ctx context.Context) error {
	err := st.conn.WithContext(ctx).Exec("SELECT 1").Error
	if err != nil {
		return fmt.Errorf("no ping in repository: %w", err)
	}
	return nil
}

func newErrOrderNotFound(id string) error {
	return fmt.Errorf("%w for number = %s", myerrors.ErrOrderNotFound, id)
}

func (st *Store) GetOrder(ctx context.Context, getOrder models.GetOrder) (*models.Order, error) {
	var order models.Order

	result := st.conn.WithContext(ctx).
		Where("order_number = ?", getOrder.Number).
		First(&order)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, newErrOrderNotFound(getOrder.Number)
		}
		return nil, fmt.Errorf("failed to get order: %w", result.Error)
	}

	return &order, nil
}

func (st *Store) SetOrder(ctx context.Context, storeOrder models.StoreOrder) error {
	order := models.Order{
		UserID:   storeOrder.UserID,
		Number:   storeOrder.Number,
		StatusID: models.OrderStatusMap[models.OrderNew],
	}

	result := st.conn.WithContext(ctx).Create(&order)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("conflict order number: %w", myerrors.ErrConflictNumber)
		}

		return fmt.Errorf("failed to save order: %w", result.Error)
	}

	return nil
}

func (st *Store) IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.Order, <-chan error) {
	ordersChan := make(chan models.Order)
	errorChan := make(chan error, 1)
	chunkSize := 1000
	var lastUploadedAt *time.Time

	go func() {
		defer close(ordersChan)
		defer close(errorChan)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				var orders []models.Order
				var query *gorm.DB

				if lastUploadedAt == nil {
					query = st.conn.
						WithContext(ctx).
						Where("user_id = ?", indexOrder.UserID).
						Order("uploaded_at asc")
				} else {
					query = st.conn.
						WithContext(ctx).
						Where("user_id = ?", indexOrder.UserID).
						Where("uploaded_at > ?", *lastUploadedAt).
						Order("uploaded_at asc")
				}

				result := query.
					Limit(chunkSize).
					Find(&orders)

				if result.Error != nil {
					errorChan <- result.Error
					return
				}

				if len(orders) == 0 {
					return
				}

				for _, order := range orders {
					select {
					case <-ctx.Done():
						return
					case ordersChan <- order:
						lastUploadedAt = order.UploadedAt
					}
				}
			}
		}
	}()

	return ordersChan, errorChan
}
