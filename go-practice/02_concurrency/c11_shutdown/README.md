# GO-C11 · Graceful shutdown сервиса с очередью

Заготовка для самостоятельной реализации. API и готовые тесты сохранены, поведенческие проверки падают с `not implemented`.

Реализуй сервис с ограниченной очередью, работниками и остановкой с бюджетом времени.

## Контракт

```go
NewService(workers, queueSize int, handle func(context.Context, int) error) (*Service, error)
(*Service).Submit(ctx context.Context, id int) error
(*Service).Shutdown(ctx context.Context) error
```

- workers положителен, queueSize допускает ноль, handle задан. Некорректная конфигурация даёт nil и ошибку. Передавай ненулевые contexts.
- Submit с nil error означает принятую работу. При полной очереди Submit ждёт место с поддержкой context. Отмена до принятия возвращает распознаваемую ошибку context.
- Начало Shutdown закрывает приём. Новые и заблокированные Submit получают ошибку, распознаваемую через errors.Is(err, ErrClosed).
- При достаточном grace period все принятые работы обрабатываются ровно один раз, Shutdown возвращает nil, работники завершаются.
- Отмена или истечение срока Shutdown отменяет оставшиеся обработчики и возвращает соответствующую ошибку context. Обработчик кооперативно реагирует на свой context.
- Повторный успешный Shutdown безопасен. После остановки Submit возвращает ErrClosed.

## Работа с готовыми тестами

1. Запусти тесты и проверь red от заглушек not implemented.
2. Реализуй Service в exercise.go до green.
3. Повтори проверки с race detector.

Готовые тесты проверяют drain при закрытом приёме, пустую очередь, отмену, deadline, повторный Shutdown, отменённый Submit и гонку Submit/Shutdown. Сигналы готовности и явные таймауты ограничивают ожидания.

TestServiceSIGTERM запускает отдельный процесс текущего тестового бинарника. Helper собирает твой Service, подтверждает готовность через pipe и принимает SIGTERM через signal.NotifyContext. Проверяются успешный выход и журнал ровно трёх принятых ID. Для запуска своего приложения используй тот же контракт остановки по SIGINT/SIGTERM.

Команды выполняются из этой папки.

```sh
go test -timeout=30s -v .
go test -timeout=30s -race -count=1 .
```
