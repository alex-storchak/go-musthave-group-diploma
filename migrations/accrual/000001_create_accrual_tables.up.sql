-- Справочник статусов для системы расчета
CREATE TABLE accrual_statuses
(
    id          INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(255)       NOT NULL
);

-- Справочник типов вознаграждений
CREATE TABLE reward_types
(
    id          INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    code        VARCHAR(10) UNIQUE NOT NULL,
    description VARCHAR(255)       NOT NULL
);

-- Зарегистрированные заказы для расчета
CREATE TABLE accrual_orders
(
    id            INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    order_number  VARCHAR(255) UNIQUE NOT NULL,
    status_id     INTEGER             NOT NULL REFERENCES accrual_statuses (id),
    accrual       DECIMAL(10, 2),
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at  TIMESTAMP WITH TIME ZONE
);

-- Товары из заказов
CREATE TABLE order_goods
(
    id          INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    order_id    INTEGER        NOT NULL REFERENCES accrual_orders (id),
    description TEXT           NOT NULL,
    price       DECIMAL(10, 2) NOT NULL
);

-- Механики вознаграждений
CREATE TABLE reward_rules
(
    id             INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    match_pattern  VARCHAR(255) UNIQUE NOT NULL,
    reward         DECIMAL(10, 2)      NOT NULL,
    reward_type_id INTEGER             NOT NULL REFERENCES reward_types (id),
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_accrual_orders_status_id ON accrual_orders (status_id);
CREATE INDEX idx_accrual_orders_number ON accrual_orders (order_number);
CREATE INDEX idx_accrual_orders_registered_at ON accrual_orders (registered_at);
CREATE INDEX idx_accrual_orders_processed_at ON accrual_orders (processed_at);
CREATE INDEX idx_order_goods_order_id ON order_goods (order_id);
CREATE INDEX idx_reward_rules_pattern ON reward_rules (match_pattern);
CREATE INDEX idx_reward_rules_type_id ON reward_rules (reward_type_id);