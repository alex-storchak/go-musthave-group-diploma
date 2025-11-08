INSERT INTO accrual_statuses (code, description)
VALUES ('REGISTERED', 'Заказ зарегистрирован, но начисление не рассчитано'),
       ('PROCESSING', 'Расчёт начисления в процессе'),
       ('INVALID', 'Заказ не принят к расчёту, вознаграждение не будет начислено'),
       ('PROCESSED', 'Расчёт начисления окончен');

INSERT INTO reward_types (code, description)
VALUES ('%', 'Процент от стоимости товара'),
       ('pt', 'Точное количество баллов');
