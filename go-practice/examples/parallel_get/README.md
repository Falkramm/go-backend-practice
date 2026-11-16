# Параллельные GET-запросы

Заготовка упражнения. В parallel_get.go определены Result и FetchURLs, функция пока возвращает nil. Исполняемые тесты предстоит написать.

```go
func FetchURLs(ctx context.Context, client *http.Client, urls []string) []Result
```

Результаты должны соответствовать порядку urls. Context и HTTP-клиент передаются вызывающим кодом. Перед реализацией определи ограничение параллелизма, обработку ошибок и отмены, затем напиши тесты с локальным httptest.Server.

Для примера проверенного ограничения параллелизма смотри [ParallelMap](../../02_concurrency/c05_parallel_map/README.md), для HTTP-запроса — [H01](../../03_http/h01_get/README.md).
