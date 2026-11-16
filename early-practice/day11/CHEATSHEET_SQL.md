# SQL-шпаргалка для собесов (Postgres)

## Каркас запроса (порядок выполнения ≠ порядок написания!)
```sql
SELECT dept, count(*) AS cnt      -- 5. что вернуть (+агрегаты)
FROM employees e                  -- 1. откуда
JOIN salaries s ON s.emp_id = e.id -- 2. соединить
WHERE e.age > 25                  -- 3. фильтр СТРОК (до группировки!)
GROUP BY dept                     -- 4. группировка
HAVING count(*) > 3               -- 4b. фильтр ГРУПП
ORDER BY cnt DESC                 -- 6. сортировка
LIMIT 10 OFFSET 20;               -- 7. срез
```

## JOIN'ы
```sql
INNER JOIN  -- только совпавшие пары (по умолчанию просто JOIN)
LEFT JOIN   -- все слева + совпавшие справа (нет пары -> NULL)
            -- «пользователи без заказов»: LEFT JOIN + WHERE o.id IS NULL
RIGHT/FULL  -- зеркально / обе стороны
```

## Классические задачки
```sql
-- Вторая по величине зарплата:
SELECT DISTINCT salary FROM emp ORDER BY salary DESC LIMIT 1 OFFSET 1;

-- Дубликаты:
SELECT name FROM emp GROUP BY name HAVING count(*) > 1;

-- Максимум в каждой группе:
SELECT dept, max(salary) FROM emp GROUP BY dept;

-- Топ-1 сотрудник каждого отдела (оконная функция):
SELECT * FROM (
  SELECT *, row_number() OVER (PARTITION BY dept ORDER BY salary DESC) AS rn
  FROM emp
) t WHERE rn = 1;
```

## Транзакции
```sql
BEGIN;  ...  COMMIT; / ROLLBACK;
BEGIN ISOLATION LEVEL REPEATABLE READ;   -- RC (дефолт) / RR / SERIALIZABLE
SELECT ... FOR UPDATE;                   -- пессимистичный замок строк
UPDATE t SET x = x + 1 WHERE id = 1;     -- атомарный инкремент
UPDATE t SET x = 5 WHERE id=1 AND version=7;  -- оптимистичный (проверить rows affected)
```
- Уровни: RC — снимок на запрос; RR — снимок на транзакцию (PG: нет фантомов,
  конкурентный UPDATE -> serialization error -> retry); Serializable — ловит write skew (SSI).
- Deadlock: детектор убьёт одну (~1с) -> retry. Профилактика: замки в одном порядке (ORDER BY id FOR UPDATE).
- MVCC: версии строк (xmin/xmax) + снимки; vacuum чистит мёртвые версии до горизонта;
  долгая транзакция = плотина для мусора ВСЕЙ базы.

## Индексы
```sql
CREATE INDEX ON users (email);                     -- B-tree (дефолт)
CREATE INDEX ON users (country, age);              -- составной: leftmost prefix!
CREATE INDEX ON users (country) INCLUDE (age);     -- покрывающий (index only scan)
CREATE INDEX ON orders (user_id) WHERE status='active';  -- частичный
CREATE INDEX ON users (lower(email));              -- функциональный
CREATE INDEX ON users (email text_pattern_ops);    -- для LIKE 'prefix%' при юникод-локали
CREATE INDEX CONCURRENTLY ...;                     -- на проде: без блокировки записи
```
- B-tree: узел = страница 8КБ, высота 3-4 на миллионы; листья связаны (диапазоны дёшевы).
- Selectivity: ≪1% -> Index Scan; ~1-10% -> Bitmap; >10-20% -> Seq Scan (индекс игнорируется!).
- Ломают индекс: функция над колонкой, LIKE '%suffix', неявный каст типов.
- Цена: каждая запись обновляет все индексы; место; bloat.
- EXPLAIN ANALYZE: Seq/Index/Index Only (Heap Fetches!)/Bitmap; actual time в мс,
  rows и time — СРЕДНЕЕ на loop; правда — в Execution Time.

## Производительность
- N+1: JOIN или WHERE id = ANY($1) пачкой.
- Пагинация: OFFSET читает и выбрасывает; keyset: WHERE id > $last ORDER BY id LIMIT n.
- sql.DB — это ПУЛ: SetMaxOpenConns (обязательно! дефолт безлимитен), SetMaxIdleConns,
  SetConnMaxLifetime. Драйвер — pgx. Сайзинг от ёмкости БД, не от хотелок.
- defer tx.Rollback() после BeginTx — всегда (no-op после Commit).
