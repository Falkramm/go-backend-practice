# GO-H02 · POST, PUT, PATCH и DELETE с JSON

Типизированный клиент заметок расположен в exercise.go. Его API использует Note, NoteInput и Client.

## Статус

Частичная реализация. Проверка 16 ноября 2026 года выявила падения при неверном baseURL, неположительном ID, ответе null, втором JSON-документе и непробельном хвосте. Существующие тесты воспроизводят эти случаи. Основная HTTP-цепочка реализована.

## Какие запросы писать

Тесты поднимают локальный HTTP-сервер и передают его адрес как baseURL. Client должен формировать следующие запросы. ID и created_at создаёт сервер. Входные данные содержат непустой title, body со строкой или null и логический pinned. Содержательную валидацию полей выполняет сервер.

| Метод | Путь | JSON запроса | Ожидаемый ответ |
|---|---|---|---|
| POST | /notes | `{"title":"Go","body":"Повторить HTTP","pinned":false}` | 201 и созданная Note |
| GET | /notes/1 | Пустое тело | 200 и Note |
| PUT | /notes/1 | `{"title":"HTTP","body":null,"pinned":true}` | 200 и полностью обновлённая Note |
| PATCH | /notes/1 | `{"body":"","pinned":false}` | 200 и Note с сохранённым title |
| DELETE | /notes/1 | Пустое тело | 204 и пустое тело |
| GET после удаления | /notes/1 | Пустое тело | 404, клиент возвращает ошибку |

Пример ответа после POST

```json
{
  "id": 1,
  "title": "Go",
  "body": "Повторить HTTP",
  "pinned": false,
  "created_at": "2026-11-09T12:00:00Z"
}
```

PUT отправляет все три изменяемых поля. PATCH отправляет только ключи changes. Отсутствующий body сохраняет прежнее значение, null очищает его, пустая строка записывает пустую строку. После приведённого PUT запрос PATCH из таблицы даёт title="HTTP", body="", pinned=false. Пустой PATCH `{}` сохраняет заметку целиком.

## API

```go
NewClient(httpClient *http.Client, baseURL string) (*Client, error)
(*Client).Create(ctx context.Context, input NoteInput) (Note, error)
(*Client).Get(ctx context.Context, id int64) (Note, error)
(*Client).Replace(ctx context.Context, id int64, input NoteInput) (Note, error)
(*Client).Patch(ctx context.Context, id int64, changes map[string]any) (Note, error)
(*Client).Delete(ctx context.Context, id int64) error
```

## Контракт

- httpClient задан с положительным Timeout. baseURL абсолютный HTTP(S) URL с необязательным prefix path и завершающим слешем, query и fragment исключены. Конфигурация проверяется в NewClient.
- Передавай ненулевой context. Для Get, Replace, Patch и Delete ID положителен, неверный ID даёт ошибку до HTTP.
- Create выполняет POST /notes и ожидает 201. Get выполняет GET /notes/{id}, Replace — PUT, Patch — PATCH, все ожидают 200. Delete выполняет DELETE /notes/{id} и ожидает 204 с пустым телом.
- Accept application/json передаётся всегда. JSON payload получает Content-Type application/json. GET и DELETE отправляются с пустым телом.
- NoteInput кодируется со всеми тремя полями title, body и pinned. Нулевые значения передаются явно, в том числе null body и pinned=false. Пустой title сервер отклоняет с 400. В Patch отсутствующий ключ сохраняется на сервере, body=null очищает поле, body="" записывает пустую строку, pinned=false передаёт false. Пустой объект PATCH допустим.
- Ответ успешного Create/Get/Replace/Patch содержит один JSON-объект Note, пробельный хвост и дополнительные поля допустимы. Пустое тело, null, массив, битый JSON и второй документ дают ошибку. Delete 204 успешно обрабатывает пустое тело.
- Каждый полученный response body закрывается при возврате.
- Неподходящий статус с полностью прочитанным телом даёт ошибку с числовым HTTP-кодом. Ошибки транспорта, context, чтения и JSON-кодирования сохраняют причины для errors.Is/As.

## Локальный контрактный этап

Запусти готовые тесты, изучи воспроизводимые ошибки и доработай клиент до green. Тесты делают реальные HTTP-запросы к локальному httptest.Server, проверяют цепочку CRUD, wire JSON и краевые ответы. Дополнительный RoundTripper контролирует Close и ошибки чтения.

```sh
go test -timeout=30s -v .
go test -timeout=30s -race -count=1 .
```

Подробный разбор таймаутов, отмены и лимита тела ответа — GO-H03.

## Этап после GO-D01

Подключи этот клиент к своему CRUD заметок на настоящей локальной PostgreSQL. Выполни create → get → replace → patch → delete → get с ожидаемым 404. Сохрани HTTP-результаты и состояние БД. Для такого прогона доступен [полный пример D01](../../examples/d01_notes/README.md).
