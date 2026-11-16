-- Задачи S1-S3. Дописывай решения и гоняй: psql -d prep -f 01_tasks.sql

-- ===== S1. Вторая по величине зарплата (решено, две версии) =====

-- твоя версия (вложенный запрос):
SELECT salary FROM (
    SELECT * FROM employees ORDER BY salary DESC LIMIT 2
) AS t
ORDER BY salary LIMIT 1;

-- идиома OFFSET + DISTINCT (устойчива к дублям максимума):
SELECT DISTINCT salary FROM employees
ORDER BY salary DESC
LIMIT 1 OFFSET 1;

-- ===== S2. Отделы, где больше 3 сотрудников, и сколько их =====
-- Конструкции: GROUP BY dept, COUNT(*), HAVING (фильтр ПО ГРУППАМ)
-- Ожидается: backend 4, frontend 4
select
dept, count(*) as count
from employees
group by dept
having count(*) > 3;
-- TODO: твой запрос сюда


-- ===== S3. Имена-дубликаты (встречаются больше 1 раза) =====
-- Ожидается: alice, bob
select
name
from employees
group by name
having count(*) > 1;

-- TODO: твой запрос сюда
