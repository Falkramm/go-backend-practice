-- ============================================================
-- LOST UPDATE ВЖИВУЮ. Два терминала: А и Б (оба psql -d prep).
-- Выполняй шаги по номерам, переключаясь между терминалами!
-- Перед стартом: psql -d prep -f 00_schema.sql (balance = 100)
-- ============================================================

-- ===== Сцена 1: аномалия (оба на дефолтном Read Committed) =====

-- [шаг 1, терминал А]
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT balance FROM accounts WHERE id = 1 FOR UPDATE;   -- видит 100

-- [шаг 2, терминал Б]
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT balance FROM accounts WHERE id = 1 FOR UPDATE;   -- тоже видит 100

-- [шаг 3, А]  «прибавили 100 в приложении» → пишем 200
UPDATE accounts SET balance = 200 WHERE id = 1;
COMMIT;

-- [шаг 4, Б]  Б тоже посчитала 100+100=200... UPDATE ПОДВИСНЕТ? Нет:
UPDATE accounts SET balance = 200 WHERE id = 1;  -- выполнится!
COMMIT;

-- [шаг 5, любой]  СКОЛЬКО ДЕНЕГ? Должно быть 300 (два раза +100):
SELECT balance FROM accounts WHERE id = 1;   -- 200. Сотня испарилась.

-- ===== Сцена 2: лечение №1 — атомарный UPDATE =====
-- Сбрось схему. Повтори шаги, но в А и Б вместо SELECT+UPDATE один запрос:
UPDATE accounts SET balance = balance + 100 WHERE id = 1;
-- Б на шаге 4 ПОДОЖДЁТ замка А (если А ещё в транзакции) и прибавит
-- к СВЕЖЕМУ значению. Итог: 300. Проверь!

-- ===== Сцена 3: лечение №2 — пессимистичный замок =====
-- Сбрось. В обеих: SELECT ... FOR UPDATE вместо голого SELECT:
-- BEGIN;
-- SELECT balance FROM accounts WHERE id = 1 FOR UPDATE;  -- Б замрёт ЗДЕСЬ
-- UPDATE accounts SET balance = 200 WHERE id = 1;
-- COMMIT;                                                -- Б проснётся и увидит 200
-- Итог: 300 (Б прочитала уже 200 и записала 300). Проверь!

-- ===== Сцена 4: лечение №3 — Repeatable Read + retry =====
-- Сбрось. В обеих BEGIN ISOLATION LEVEL REPEATABLE READ; и шаги 1-4 как в сцене 1.
-- На шаге 4 Б получит:
--   ERROR: could not serialize access due to concurrent update
-- → транзакция Б мертва, ROLLBACK и повторить заново = ретрай. Проверь!
