INSERT INTO balance_operation_types (code, description)
VALUES ('ACCRUAL', 'Начисление баллов за заказ'),
       ('WITHDRAWAL', 'Списание баллов для оплаты заказа');

INSERT INTO order_statuses (code, description)
VALUES ('NEW', 'Заказ загружен в систему, но не попал в обработку'),
       ('PROCESSING', 'Вознаграждение за заказ рассчитывается'),
       ('INVALID', 'Система расчёта вознаграждений отказала в расчёте'),
       ('PROCESSED', 'Данные по заказу проверены и информация о расчёте успешно получена');