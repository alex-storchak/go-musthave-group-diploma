-- Пользователи системы
CREATE TABLE IF NOT EXISTS users
(
    id            INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    login         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255)        NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

DROP TYPE IF EXISTS order_status;
CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

-- Заказы пользователей
CREATE TABLE IF NOT EXISTS orders
(
    order_number      VARCHAR(255)   PRIMARY KEY,
    user_id           INTEGER        NOT NULL,
    status            order_status,
    accrual           DECIMAL(10, 2),
    uploaded_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at      TIMESTAMP WITH TIME ZONE
);

-- История операций с балансом
CREATE TABLE IF NOT EXISTS withdrawals
(
    order_number      VARCHAR(255) PRIMARY KEY,
    user_id           INTEGER        NOT NULL,
    sum               DECIMAL(10, 2) NOT NULL,
    processed_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Баланс пользователей
CREATE TABLE IF NOT EXISTS balance
(
    user_id         INTEGER PRIMARY KEY,
    current         DECIMAL(10, 2)           DEFAULT 0,
    total_withdrawn DECIMAL(10, 2)           DEFAULT 0,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_uploaded_at ON orders (uploaded_at);
CREATE INDEX idx_balance_operations_user_id ON withdrawals (user_id);
CREATE INDEX idx_balance_operations_processed_at ON withdrawals (processed_at);