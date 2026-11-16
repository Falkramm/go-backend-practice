# Практика Go

Здесь собраны упражнения по ошибкам, JSON, конкурентности, HTTP и PostgreSQL. В каждой папке находятся условие, код и проверки. Полный пример сервиса заметок находится в [examples/d01_notes](examples/d01_notes/README.md).

## Быстрый старт

Понадобится Go 1.24 или новее. Из этой папки запусти проверенные упражнения с race detector.

```sh
go test -race -count=1 -timeout=30s \
  ./01_errors_json/... ./02_concurrency/c0* \
  ./02_concurrency/c10_token_bucket ./03_http/h01_get ./internal/...
```

Для отдельного упражнения перейди в его папку и выполни `go test -v .`. Для запуска CRUD подготовь PostgreSQL по [инструкции готового примера](examples/d01_notes/README.md).

## Упражнения и состояние

Состояние сверено с кодом и проверками 16 ноября 2026 года. C01–C10 и H01 прошли тесты с `-race`. E01 и J01 прошли имеющиеся тесты, дополнительные сценарии описаны в условиях. Полный пример D01 проверен на отдельном PostgreSQL 18.6 с `-race` и `go vet`.

| Код | Тема | Состояние |
| --- | --- | --- |
| [E01](01_errors_json/e01_errors/README.md) | Ошибки, `%w`, `errors.Is` и `errors.As` | Реализация и базовые тесты |
| [J01](01_errors_json/j01_json/README.md) | JSON-теги, кодирование и декодирование | Реализация и round-trip тесты |
| [C01](02_concurrency/c01_alternation/README.md) | Чередование горутин | Тесты и `-race` пройдены |
| [C02](02_concurrency/c02_producer_consumer/README.md) | Producer/consumer и context | Тесты и `-race` пройдены |
| [C03](02_concurrency/c03_fan_in/README.md) | Fan-in | Тесты и `-race` пройдены |
| [C04](02_concurrency/c04_worker_pool/README.md) | Worker pool с ограниченной очередью | Тесты и `-race` пройдены |
| [C05](02_concurrency/c05_parallel_map/README.md) | Параллельный map с сохранением порядка | Тесты и `-race` пройдены |
| [C06](02_concurrency/c06_pipeline/README.md) | Pipeline с fan-out/fan-in | Тесты и `-race` пройдены |
| [C07](02_concurrency/c07_batch_processor/README.md) | Пачки по размеру и таймеру | Тесты и `-race` пройдены |
| [C08](02_concurrency/c08_counter/README.md) | Конкурентный счётчик и snapshot | Тесты и `-race` пройдены |
| [C09](02_concurrency/c09_rate_limiter/README.md) | Rate limiter | Тесты и `-race` пройдены |
| [C10](02_concurrency/c10_token_bucket/README.md) | Token bucket по ключам | Тесты и `-race` пройдены |
| [C11](02_concurrency/c11_shutdown/README.md) | Graceful shutdown очереди | Заготовка, поведенческие тесты падают |
| [H01](03_http/h01_get/README.md) | GET, query, заголовки и JSON | Тесты и `-race` пройдены |
| [H02](03_http/h02_json_methods/README.md) | Типизированный JSON HTTP-клиент | Частично, проверки URL, ID и строгого JSON падают |
| [H03](03_http/h03_timeouts/README.md) | Таймауты и отмена HTTP | Условие и заготовки, проверки предстоит написать |
| [D01](04_crud/d01_notes/README.md) | CRUD заметок | Частичная реализация, схема и запуск не завершены |
| [D02](04_crud/d02_catalog/README.md) | Каталог, JOIN и UPSERT | Частичная реализация, ошибка сборки |
| [D03](04_crud/d03_orders/README.md) | Заказы, транзакции и конкуренция | Частичная реализация, ошибка сборки |

## Примеры и вспомогательный код

- [Полный CRUD заметок](examples/d01_notes/README.md) содержит HTTP-обработчики, SQL-схему, запуск и проверку существующими контрактными тестами.
- [Параллельные GET-запросы](examples/parallel_get/README.md) содержат сигнатуру `FetchURLs` и заготовку реализации.
- `internal/clock` и `internal/testclock` дают реальные и управляемые часы для проверок таймеров.
- `04_crud/internal/testdb` и локальный помощник D01 создают временные схемы PostgreSQL с автоматическим удалением.

## Проверка всего модуля

```sh
go test -count=1 -timeout=30s ./...
```

Команда показывает и незавершённые задания. C11, H02 и сервер D01 дают ожидаемые для текущих черновиков падения. D02 вызывает отсутствующий `handler.New`, D03 содержит неиспользуемую переменную `err`, поэтому общая сборка завершается ошибкой. PostgreSQL-тесты запускаются при заданном `TEST_DATABASE_URL` или `DATABASE_URL`, иначе пропускаются.

Модуль использует `example.com/go-practice`, `database/sql` и драйвер `github.com/jackc/pgx/v5/stdlib` версии `v5.7.6`.
