# GO-H01 · HTTP GET — параметры, заголовки и JSON

Реализация прошла тесты и проверку `-race` 16 ноября 2026 года.

GetItems получает список Item через переданный переиспользуемый http.Client.

## API

```go
type Item struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func GetItems(ctx context.Context, client *http.Client, endpoint string,
    query url.Values, requestID string, maxBodyBytes int64) ([]Item, error)
```

## Контракт

- Передавай ненулевой context. client задан, его Timeout положителен, maxBodyBytes положителен. Неверная конфигурация и относительный endpoint дают ошибку до HTTP-запроса.
- Выполни GET по абсолютному endpoint, сохрани его path. query.Encode заменяет query в endpoint. Входной url.Values и настройки client сохраняются.
- Заголовки запроса — Accept application/json и X-Request-ID с переданным requestID.
- Статус 200…299 содержит один JSON-массив Item. Пустой массив и пробельный хвост допустимы. Пустое тело, null, объект, битый JSON, второй документ и непробельный хвост дают ошибку.
- Тело размером ровно maxBodyBytes допустимо. Прочитай максимум maxBodyBytes+1 байт с учётом переполнения int64 на максимальном лимите. Превышение на успешном статусе возвращает ошибку.
- Полностью прочитанный ответ с другим HTTP-статусом возвращает ошибку с числовым статусом. При ошибке чтения возвращается её причина.
- Каждый полученный response body закрывается при возврате. Ошибки транспорта, context и чтения сохраняют причину для errors.Is.

## Работа с готовыми тестами

Реализация находится в `exercise.go`, проверки — в `exercise_test.go`.

Основные сценарии выполняют настоящие запросы к httptest.Server. Дополнительный RoundTripper проверяет закрытие тела, бюджет чтения и причины ошибок. Отдельные сценарии таймаутов и отмены описаны в [H03](../h03_timeouts/README.md).

Команды выполняются из этой папки.

```sh
go test -timeout=30s -v .
go test -timeout=30s -race -count=1 .
```
