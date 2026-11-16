-- День 11: миллион пользователей для экспериментов с индексами.
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id         serial PRIMARY KEY,
    email      text NOT NULL,
    country    text NOT NULL,
    age        int  NOT NULL,
    created_at timestamptz NOT NULL
);

INSERT INTO users (email, country, age, created_at)
SELECT
    'user' || i || '@example.com',
    (ARRAY['ru','kz','by','de','us'])[1 + i % 5],
    18 + i % 60,
    now() - (i % 365) * interval '1 day'
FROM generate_series(1, 1_000_000) AS i;

ANALYZE users;  -- собрать статистику для планировщика
