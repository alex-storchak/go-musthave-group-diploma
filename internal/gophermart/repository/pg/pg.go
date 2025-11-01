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
	"strings"
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

func (st *Store) GetNewOrders(ctx context.Context, orders []models.OrderProcess) ([]models.OrderProcess, error) {

	var numbers []string
	for _, order := range orders {
		numbers = append(numbers, order.Number)
	}

	str := ""
	if len(numbers) != 0 {
		ns := "'" + strings.Join(numbers, "','") + "'"
		str = fmt.Sprintf("AND order_number NOT IN (%s)", ns)
	}

	var result []models.OrderProcess

	err := st.conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := tx.WithContext(ctx).Raw(fmt.Sprintf(`
			WITH selected_orders AS (
				SELECT order_number
				FROM orders
				WHERE status IN ('NEW', 'PROCESSING')
				%s
				ORDER BY uploaded_at ASC
				LIMIT 100
				FOR UPDATE
			),
			updated_orders AS (
				UPDATE orders
				SET
					status = 'PROCESSING',
					processed_at = CASE
						WHEN status = 'NEW' THEN NOW()
						ELSE processed_at
					END
				WHERE order_number IN (SELECT order_number FROM selected_orders)
				RETURNING order_number, processed_at
			)
			SELECT order_number FROM updated_orders ORDER BY processed_at ASC;
		`, str)).Rows()

		if err != nil {
			return fmt.Errorf("run sql: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var order models.OrderProcess
			if err = rows.Scan(&order.Number); err != nil {
				return fmt.Errorf("scan order: %w", err)
			}
			result = append(result, order)
		}

		if err = rows.Err(); err != nil {
			return fmt.Errorf("edit result %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
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

func (st *Store) UpdateOrderInvalid(ctx context.Context, accrualResponse *models.AccrualResponse) error {
	res := st.conn.
		WithContext(ctx).
		Table("balance").
		Where("order_number = ?", accrualResponse.Number).
		Updates(map[string]interface{}{
			"status":       models.OrderInvalid,
			"processed_at": time.Now(),
		})
	if res.Error != nil {
		return fmt.Errorf("update order invalid: %w", res.Error)
	}

	return nil
}

func (st *Store) UpdateOrderProcessed(ctx context.Context, accrualResponse *models.AccrualResponse) error {
	return st.conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Настройка изоляции транзакции
		if err := setupTransactionIsolation(tx); err != nil {
			return err
		}

		// 2. Валидация входных данных
		if err := validateAccrualResponse(accrualResponse); err != nil {
			return err
		}

		// 3. Получение заказа
		order, err := getOrderByNumber(tx, ctx, accrualResponse.Number)
		if err != nil {
			return err
		}

		// 4. Обновление баланса пользователя
		if err := updateUserBalance(tx, ctx, order.UserID, accrualResponse.Accrual); err != nil {
			return err
		}

		// 5. Обновление статуса заказа
		if err := markOrderAsProcessed(tx, ctx, accrualResponse); err != nil {
			return err
		}

		return nil
	})
}

func setupTransactionIsolation(tx *gorm.DB) error {
	if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
		return fmt.Errorf("transaction level: %w", err)
	}
	return nil
}

func validateAccrualResponse(resp *models.AccrualResponse) error {
	if resp == nil {
		return myerrors.ErrAccrualResponseNil
	}
	if resp.Number == "" {
		return myerrors.ErrOrderNumberNil
	}
	return nil
}

func getOrderByNumber(tx *gorm.DB, ctx context.Context, number string) (models.Order, error) {
	var order models.Order
	result := tx.WithContext(ctx).
		Table("orders").
		Where("order_number = ?", number).
		First(&order)

	if result.Error != nil {
		return models.Order{}, fmt.Errorf("get order by number: %w", result.Error)
	}
	return order, nil
}

func updateUserBalance(tx *gorm.DB, ctx context.Context, userID models.UserID, accrual models.RoundedFloat64) error {
	var balance models.ShowBalanceResponse
	res := tx.
		WithContext(ctx).
		Table("balance").
		Where("user_id = ?", userID).
		First(&balance)

	if res.RowsAffected < 1 {
		// Создаём баланс, если его нет
		newBalance := models.Balance{
			UserID:         userID,
			Current:        accrual,
			TotalWithdrawn: 0,
		}
		res = tx.WithContext(ctx).Table("balance").Create(&newBalance)
		if res.Error != nil {
			return fmt.Errorf("create default balance: %w", res.Error)
		}
	} else {
		// Обновляем существующий баланс
		res = tx.
			WithContext(ctx).
			Table("balance").
			Where("user_id = ?", userID).
			Updates(map[string]interface{}{
				"current":    gorm.Expr("current + ?", accrual),
				"updated_at": time.Now(),
			})
		if res.Error != nil {
			return fmt.Errorf("update balance: %w", res.Error)
		}
	}
	return nil
}

func markOrderAsProcessed(tx *gorm.DB, ctx context.Context, resp *models.AccrualResponse) error {
	res := tx.
		WithContext(ctx).
		Table("orders").
		Where("order_number = ?", resp.Number).
		Updates(map[string]interface{}{
			"status":       models.OrderProcessed,
			"processed_at": time.Now(),
			"accrual":      resp.Accrual,
		})

	if res.Error != nil {
		return fmt.Errorf("update order processed: %w", res.Error)
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

func (st *Store) IndexOrder(ctx context.Context, indexOrder models.IndexOrder) ([]models.IndexOrderResponse, error) {
	var orders []models.IndexOrderResponse

	err := st.conn.
		WithContext(ctx).
		Table("orders").
		Where("user_id = ?", indexOrder.UserID).
		Order("uploaded_at asc").
		Find(&orders).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	return orders, nil
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

func (st *Store) StoreWithdrawal(
	ctx context.Context,
	storeWithdrawal models.StoreWithdrawal,
	setDefaultBalance models.SetDefaultBalanceRequest,
) error {
	return st.conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Настройка изоляции транзакции
		if err := setupTransactionIsolation(tx); err != nil {
			return err
		}

		// 2. Получение текущего баланса пользователя
		balance, err := getUserBalance(tx, ctx, storeWithdrawal.UserID, setDefaultBalance)
		if err != nil {
			return err
		}

		// 3. Проверка достаточности средств
		if balance.Current < storeWithdrawal.Sum {
			return myerrors.ErrBalance
		}

		// 4. Обновление баланса (списание)
		if err := deductWithdrawalAmount(tx, ctx, storeWithdrawal); err != nil {
			return err
		}

		// 5. Создание записи о выводе
		if err := createWithdrawalRecord(tx, ctx, storeWithdrawal); err != nil {
			return err
		}

		return nil
	})
}

func getUserBalance(tx *gorm.DB, ctx context.Context, userID models.UserID, setDefaultBalance models.SetDefaultBalanceRequest) (models.ShowBalanceResponse, error) {
	var balance models.ShowBalanceResponse
	result := tx.
		WithContext(ctx).
		Table("balance").
		Where("user_id = ?", userID).
		First(&balance)

	if result.Error == nil {
		return balance, nil
	}

	// Если баланс не найден — создаём дефолтный
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		defaultBalance := models.Balance{
			UserID:         userID,
			Current:        setDefaultBalance.Current,
			TotalWithdrawn: setDefaultBalance.TotalWithdrawn,
		}

		createResult := tx.
			WithContext(ctx).
			Table("balance").
			Create(&defaultBalance)

		if createResult.Error != nil {
			return models.ShowBalanceResponse{}, fmt.Errorf(
				"create default balance: %w",
				createResult.Error,
			)
		}

		balance.Current = setDefaultBalance.Current
		balance.TotalWithdrawn = setDefaultBalance.TotalWithdrawn
		return balance, nil
	}

	return models.ShowBalanceResponse{}, fmt.Errorf("get balance: %w", result.Error)
}

func deductWithdrawalAmount(
	tx *gorm.DB,
	ctx context.Context,
	withdrawal models.StoreWithdrawal,
) error {
	result := tx.
		WithContext(ctx).
		Table("balance").
		Where("user_id = ?", withdrawal.UserID).
		Updates(map[string]interface{}{
			"current":         gorm.Expr("current - ?", withdrawal.Sum),
			"total_withdrawn": gorm.Expr("total_withdrawn + ?", withdrawal.Sum),
		})

	if result.Error != nil {
		return fmt.Errorf("update balance: %w", result.Error)
	}
	return nil
}

func createWithdrawalRecord(
	tx *gorm.DB,
	ctx context.Context,
	withdrawal models.StoreWithdrawal,
) error {
	result := tx.
		WithContext(ctx).
		Table("withdrawals").
		Create(&withdrawal)

	if result.Error != nil {
		return fmt.Errorf("create withdrawal: %w", result.Error)
	}
	return nil
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

func (st *Store) IndexWithdrawal(ctx context.Context, indexWithdrawal models.IndexWithdrawal) ([]models.IndexWithdrawalResponse, error) {
	var withdrawals []models.IndexWithdrawalResponse

	err := st.conn.
		WithContext(ctx).
		Table("withdrawals").
		Where("user_id = ?", indexWithdrawal.UserID).
		Order("processed_at asc").
		Find(&withdrawals).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to query withdrawals: %w", err)
	}

	return withdrawals, nil
}
