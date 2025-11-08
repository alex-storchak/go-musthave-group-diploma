DROP INDEX IF EXISTS idx_reward_rules_type_id;
DROP INDEX IF EXISTS idx_reward_rules_pattern;
DROP INDEX IF EXISTS idx_order_goods_order_id;
DROP INDEX IF EXISTS idx_accrual_orders_processed_at;
DROP INDEX IF EXISTS idx_accrual_orders_registered_at;
DROP INDEX IF EXISTS idx_accrual_orders_number;
DROP INDEX IF EXISTS idx_accrual_orders_status_id;

DROP TABLE IF EXISTS reward_rules;
DROP TABLE IF EXISTS order_goods;
DROP TABLE IF EXISTS accrual_orders;
DROP TABLE IF EXISTS reward_types;
DROP TABLE IF EXISTS accrual_statuses;