# Каналы, горутины и жизненный цикл

C01–C10 реализованы и прошли проверки с race detector 16 ноября 2026 года. C11 содержит контракт, готовые тесты и заготовку сервиса. Его поведенческие тесты падают с `not implemented`.

## Упражнения

| Код | Тема | Условие и API |
| --- | --- | --- |
| C01 | Чередование двух горутин | [c01_alternation](c01_alternation/README.md) |
| C02 | Producer/consumer с context | [c02_producer_consumer](c02_producer_consumer/README.md) |
| C03 | Fan-in — объединить несколько каналов | [c03_fan_in](c03_fan_in/README.md) |
| C04 | Worker pool с ограниченной очередью | [c04_worker_pool](c04_worker_pool/README.md) |
| C05 | Параллельный map с сохранением порядка | [c05_parallel_map](c05_parallel_map/README.md) |
| C06 | Pipeline — три стадии с fan-out/fan-in | [c06_pipeline](c06_pipeline/README.md) |
| C07 | Batch processor — пачка по размеру или таймеру | [c07_batch_processor](c07_batch_processor/README.md) |
| C08 | Конкурентный счётчик по ключам и snapshot | [c08_counter](c08_counter/README.md) |
| C09 | Простой rate limiter без накопления burst | [c09_rate_limiter](c09_rate_limiter/README.md) |
| C10 | Token bucket по ключам и время ожидания | [c10_token_bucket](c10_token_bucket/README.md) |
| C11 | Graceful shutdown сервиса с очередью | [c11_shutdown](c11_shutdown/README.md) |

## Запуск

Из корня Go-проекта выбери пакет упражнения.

```sh
go test -timeout=30s -v ./02_concurrency/c03_fan_in
go test -race -count=1 -timeout=30s ./02_concurrency/c03_fan_in
```

Проверить реализованные упражнения можно одной командой из корня модуля.

```sh
go test -race -count=1 -timeout=30s ./02_concurrency/c0* ./02_concurrency/c10_token_bucket
```

## Время в C07, C09 и C10

Параметр `clock.Clock` задаёт источник времени. Для обычного запуска используй `clock.Real{}` из `example.com/go-practice/internal/clock`. Готовые тесты используют `internal/testclock` с явным `Advance`. В C07 и C09 сценарии ожидания используют барьер `WaitForTimers` и короткие проверки с реальными таймерами. В C10 время продвигается между вызовами `Access`.

Общие часы, обработка пачек и лимитеры реализованы. Управляемое время позволяет воспроизводить граничные моменты без ожидания реальных интервалов.

## Shutdown в C11

Кроме проверки API есть тест дочернего процесса. Он передаёт `SIGTERM`, ждёт завершения и сверяет журнал обработанных ID. Все данные процесса находятся в каталоге `t.TempDir`.

## Проверка внутреннего устройства

README C03 и C06 отмечают свойства, которые требуют ревью реализации. В C03 чтение выхода ради проверки закрытия способно разблокировать ошибочную отправку. В C06 число внутренних работников и устройство стадий проверяются вместе с кодом.
