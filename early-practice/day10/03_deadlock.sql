-- ============================================================
-- DEADLOCK ВЖИВУЮ. Два терминала: А и Б. Перед стартом — 00_schema.
-- ============================================================

-- ===== Сцена 1: поймать deadlock =====

-- [шаг 1, А]  лочим счёт 1
BEGIN;
UPDATE accounts SET balance = balance - 10 WHERE id = 1;

-- [шаг 2, Б]  лочим счёт 2
BEGIN;
UPDATE accounts SET balance = balance - 10 WHERE id = 2;

-- [шаг 3, А]  хотим счёт 2 → он у Б → А ВИСНЕТ (это нормально, жди)
UPDATE accounts SET balance = balance + 10 WHERE id = 2;

-- [шаг 4, Б]  хотим счёт 1 → он у А → кольцо!
UPDATE accounts SET balance = balance + 10 WHERE id = 1;
-- Через ~1 секунду (deadlock_timeout) детектор убьёт ОДНУ из транзакций:
--   ERROR: deadlock detected
--   DETAIL: Process ... waits for ShareLock on transaction ...
-- Вторая — доживёт. Жертве: ROLLBACK; и ретраить.

-- [шаг 5]  Заверши выжившую (COMMIT;), сбрось схему.

-- ===== Сцена 2: профилактика — одинаковый порядок замков =====
-- Обе транзакции сначала лочат ОБА счёта ПО ВОЗРАСТАНИЮ id:
-- BEGIN;
-- SELECT * FROM accounts WHERE id IN (1,2) ORDER BY id FOR UPDATE;
--   (Б замрёт на этой строчке, ждя А, — но НЕ кольцом, а очередью!)
-- UPDATE accounts SET balance = balance - 10 WHERE id = 1;
-- UPDATE accounts SET balance = balance + 10 WHERE id = 2;
-- COMMIT;
-- Deadlock геометрически невозможен: очередь вместо кольца. Проверь!
