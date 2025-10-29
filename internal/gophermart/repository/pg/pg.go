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
func newErrBalanceNotFound(id models.UserID) error {
	return fmt.Errorf("%w for user_id = %d", myerrors.ErrBalanceNotFound, id)
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

func (st *Store) CountOrder(ctx context.Context, indexOrder models.IndexOrder) (int64, error) {
	var count int64
	res := st.conn.
		WithContext(ctx).
		Table("orders").
		Where("user_id = ?", indexOrder.UserID).Count(&count)
	if res.Error != nil {
		return 0, fmt.Errorf("count orders: %w", res.Error)
	}

	return count, nil
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

	result := st.conn.
		WithContext(ctx).
		Table("balance").
		Where("user_id = ?", getBalance.UserID).
		First(&balance)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, newErrBalanceNotFound(getBalance.UserID)
		}
		return nil, fmt.Errorf("get balance: %w", result.Error)
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
		WithContext(ctx).
		Table("balance").
		Create(&balance)

	if result.Error != nil {
		return fmt.Errorf("create default balance: %w", result.Error)
	}

	return nil
}

func (st *Store) StoreWithdrawal(ctx context.Context, storeWithdrawal models.StoreWithdrawal, setDefaultBalance models.SetDefaultBalanceRequest) error {
	return st.conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Устанавливаем уровень изоляции
		if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
			return err
		}

		var order models.Order

		o := tx.WithContext(ctx).
			Where("order_number = ?", storeWithdrawal.Number).
			Where("user_id = ?", storeWithdrawal.UserID).
			First(&order)

		if o.Error != nil {
			if errors.Is(o.Error, gorm.ErrRecordNotFound) {
				return newErrOrderNotFound(storeWithdrawal.Number)
			}
			return fmt.Errorf("failed to get order: %w", o.Error)
		}

		var balance models.ShowBalanceResponse

		b := tx.WithContext(ctx).
			Table("balance").
			Where("user_id = ?", storeWithdrawal.UserID).
			First(&balance)

		if b.Error != nil {
			if errors.Is(b.Error, gorm.ErrRecordNotFound) {
				balanceDefault := models.Balance{
					UserID:         setDefaultBalance.UserID,
					Current:        setDefaultBalance.Current,
					TotalWithdrawn: setDefaultBalance.TotalWithdrawn,
				}

				bd := tx.
					WithContext(ctx).
					Table("balance").
					Create(&balanceDefault)

				if bd.Error != nil {
					return fmt.Errorf("create default balance: %w", b.Error)
				}

				balance.Current = setDefaultBalance.Current
				balance.TotalWithdrawn = setDefaultBalance.TotalWithdrawn
			}
			return fmt.Errorf("get balance: %w", b.Error)
		}

		if balance.Current < storeWithdrawal.Sum {
			return myerrors.ErrBalance
		}

		res := tx.
			WithContext(ctx).
			Table("balance").
			Where("user_id = ?", storeWithdrawal.UserID).
			Updates(map[string]interface{}{
				"current":         gorm.Expr("current - ?", storeWithdrawal.Sum),
				"total_withdrawn": gorm.Expr("total_withdrawn + ?", storeWithdrawal.Sum),
			})
		if res.Error != nil {
			return fmt.Errorf("update balance: %w", res.Error)
		}

		res = tx.
			WithContext(ctx).
			Table("withdrawals").
			Create(&storeWithdrawal)
		if res.Error != nil {
			return fmt.Errorf("create withdrawal: %w", res.Error)
		}

		return nil
	})
}

func (st *Store) CountWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (int64, error) {
	var count int64
	res := st.conn.
		WithContext(ctx).
		Table("withdrawals").
		Where("user_id = ?", indexWithdrawal.UserID).Count(&count)
	if res.Error != nil {
		return 0, fmt.Errorf("count withdrawal: %w", res.Error)
	}

	return count, nil
}

func (st *Store) IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) (<-chan models.IndexWithdrawalResponse, <-chan error) {
	withdrawalChan := make(chan models.IndexWithdrawalResponse)
	errorChan := make(chan error, 1)
	chunkSize := 1000
	var lastProcessedAt *time.Time

	go func() {
		defer close(withdrawalChan)
		defer close(errorChan)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				var withdrawals []models.IndexWithdrawalResponse
				var query *gorm.DB

				if lastProcessedAt == nil {
					query = st.conn.
						WithContext(ctx).
						Table("withdrawals").
						Where("user_id = ?", indexWithdrawal.UserID).
						Order("processed_at asc")
				} else {
					query = st.conn.
						WithContext(ctx).
						Table("withdrawals").
						Where("user_id = ?", indexWithdrawal.UserID).
						Where("processed_at > ?", *lastProcessedAt).
						Order("processed_at asc")
				}

				result := query.
					Limit(chunkSize).
					Find(&withdrawals)

				if result.Error != nil {
					errorChan <- result.Error
					return
				}

				if len(withdrawals) == 0 {
					return
				}

				for _, withdrawal := range withdrawals {
					select {
					case <-ctx.Done():
						return
					case withdrawalChan <- withdrawal:
						lastProcessedAt = withdrawal.ProcessedAt
					}
				}
			}
		}
	}()

	return withdrawalChan, errorChan
}
