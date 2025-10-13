-- Справочник типов операций с балансом
CREATE TABLE balance_operation_types
(
    id          INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(255)       NOT NULL
);

-- Справочник статусов заказов для лояльности
CREATE TABLE order_statuses
(
    id          INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(255)       NOT NULL
);

-- Пользователи системы
CREATE TABLE users
(
    id            INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    login         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255)        NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Заказы пользователей
CREATE TABLE orders
(
    id           INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id      INTEGER             NOT NULL REFERENCES users (id),
    order_number VARCHAR(255) UNIQUE NOT NULL,
    status_id    INTEGER             NOT NULL REFERENCES order_statuses (id),
    uploaded_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- История операций с балансом
CREATE TABLE balance_operations
(
    id                INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id           INTEGER        NOT NULL REFERENCES users (id),
    operation_type_id INTEGER        NOT NULL REFERENCES balance_operation_types (id),
    order_number      VARCHAR(255),
    amount            DECIMAL(10, 2) NOT NULL,
    processed_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    description       TEXT
);

-- Баланс пользователей
CREATE TABLE balance
(
    user_id         INTEGER PRIMARY KEY REFERENCES users (id),
    current         DECIMAL(10, 2)           DEFAULT 0,
    total_accrued   DECIMAL(10, 2)           DEFAULT 0,
    total_withdrawn DECIMAL(10, 2)           DEFAULT 0,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_status_id ON orders (status_id);
CREATE INDEX idx_orders_uploaded_at ON orders (uploaded_at);
CREATE INDEX idx_balance_operations_user_id ON balance_operations (user_id);
CREATE INDEX idx_balance_operations_type_id ON balance_operations (operation_type_id);
CREATE INDEX idx_balance_operations_processed_at ON balance_operations (processed_at);