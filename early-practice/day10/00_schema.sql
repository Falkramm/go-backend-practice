-- Схема дня 10. Идемпотентно: можно перезапускать сколько угодно.
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS accounts;

CREATE TABLE employees (
    id     serial PRIMARY KEY,
    name   text NOT NULL,
    dept   text NOT NULL,
    salary int  NOT NULL
);

INSERT INTO employees (name, dept, salary) VALUES
 ('alice','backend',300), ('bob','backend',250), ('carol','backend',250),
 ('dave','backend',180), ('eve','frontend',220), ('frank','frontend',200),
 ('grace','frontend',210), ('heidi','frontend',190), ('ivan','ops',170),
 ('judy','ops',160), ('bob','ops',150), ('alice','data',280), ('mallory','data',240);

CREATE TABLE accounts (
    id      int PRIMARY KEY,
    balance int NOT NULL
);

INSERT INTO accounts VALUES (1, 100), (2, 100);
