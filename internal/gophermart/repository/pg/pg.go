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
func newErrBalanceNotFound(id int64) error {
	return fmt.Errorf("%w for user_id = %s", myerrors.ErrBalanceNotFound, id)
}

func (st *Store) GetOrderUser(ctx context.Context, getOrderUser models.GetOrderUser) (*models.Order, error) {
	var order models.Order

	result := st.conn.WithContext(ctx).
		Where("order_number = ?", getOrderUser.Number).
		Where("user_id = ?", getOrderUser.UserID).
		First(&order)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, newErrOrderNotFound(getOrderUser.Number)
		}
		return nil, fmt.Errorf("failed to get order: %w", result.Error)
	}

	return &order, nil
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
		UserID: storeOrder.UserID,
		Number: storeOrder.Number,
		Status: models.OrderNew,
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

func (st *Store) IndexOrder(ctx context.Context, indexOrder models.IndexOrder) (<-chan models.IndexOrderResponse, <-chan error) {
	ordersChan := make(chan models.IndexOrderResponse)
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
				var orders []models.IndexOrderResponse
				var query *gorm.DB

				if lastUploadedAt == nil {
					query = st.conn.
						WithContext(ctx).
						Table("orders").
						Where("user_id = ?", indexOrder.UserID).
						Order("uploaded_at asc")
				} else {
					query = st.conn.
						WithContext(ctx).
						Table("orders").
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

func (st *Store) GetBalance(ctx context.Context, getBalance models.GetBalanceRequest) (*models.ShowBalanceResponse, error) {
	var balance models.ShowBalanceResponse

	result := st.conn.WithContext(ctx).
		Table("balance").
		Where("user_id = ?", getBalance.UserID).
		First(&balance)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, newErrBalanceNotFound(getBalance.UserID)
		}
		return nil, fmt.Errorf("get order: %w", result.Error)
	}

	return &balance, nil
}

func (st *Store) SetDefaultBalance(ctx context.Context, setDefaultBalance models.SetDefaultBalanceRequest) error {
	balance := models.Balance{
		UserID:         setDefaultBalance.UserID,
		Current:        setDefaultBalance.Current,
		TotalWithdrawn: setDefaultBalance.TotalWithdrawn,
	}

	result := st.conn.
		Table("balance").
		WithContext(ctx).
		Create(&balance)

	if result.Error != nil {
		return fmt.Errorf("create default balance: %w", result.Error)
	}

	return nil
}
