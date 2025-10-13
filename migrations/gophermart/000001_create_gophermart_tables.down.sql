DROP INDEX IF EXISTS idx_balance_operations_processed_at;
DROP INDEX IF EXISTS idx_balance_operations_type_id;
DROP INDEX IF EXISTS idx_balance_operations_user_id;
DROP INDEX IF EXISTS idx_orders_uploaded_at;
DROP INDEX IF EXISTS idx_orders_status_id;
DROP INDEX IF EXISTS idx_orders_user_id;

DROP TABLE IF EXISTS balance;
DROP TABLE IF EXISTS balance_operations;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS order_statuses;
DROP TABLE IF EXISTS balance_operation_types;