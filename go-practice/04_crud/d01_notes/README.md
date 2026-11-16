# GO-D01 · CRUD заметок — HTTP, JSON и PostgreSQL

Частичная реализация. Написаны три базовых метода, ReplaceNote, PatchNote и DeleteNote содержат заглушки. SQL-схема и серверный запуск остаются заготовками. Тесты HTTP-таймаутов и остановки сервера падают. PostgreSQL-сценарии требуют завершения схемы и обработчиков.

Полная реализация с тем же контрактом находится в [готовом примере](../../examples/d01_notes/README.md). Его можно запустить и проверить через examples/d01_notes/check.py.

## Модель и таблица

Таблица notes содержит следующие столбцы.

| Столбец | Контракт |
|---|---|
| id | BIGSERIAL, первичный ключ |
| title | TEXT NOT NULL, непустое после trim |
| body | TEXT, допускает NULL и пустую строку |
| pinned | BOOLEAN NOT NULL DEFAULT false |
| created_at | TIMESTAMPTZ NOT NULL DEFAULT now() |

Заполни schema.sql. Используй обычные имена таблиц, тесты применяют файл внутри своей отдельной схемы.

Пример ответа сервера

```json
{
  "id": 1,
  "title": "Go",
  "body": "Повторить HTTP",
  "pinned": false,
  "created_at": "2026-11-09T12:00:00Z"
}
```

ID и created_at создаются при INSERT. PUT и PATCH сохраняют их значения. Заголовок проверяется через TrimSpace на непустоту, исходная строка сохраняется.

## HTTP

| Запрос | Тело | Успех |
|---|---|---|
| POST /notes | {"title":"Go","body":"Повторить HTTP","pinned":false} | 201, Note и Location /notes/{id} |
| GET /notes/{id} | Пустое | 200 и Note |
| GET /notes?limit=20&offset=0 | Пустое | 200 и массив Note по id ASC |
| PUT /notes/{id} | {"title":"HTTP","body":null,"pinned":true} | 200 и заменённая Note |
| PATCH /notes/{id} | {"body":"","pinned":false} | 200 и изменённая Note |
| DELETE /notes/{id} | Пустое | 204 и пустое тело |

У списка по умолчанию limit=20, offset=0. Допустимы limit от 1 до 100 и offset>=0. Пустой список возвращается как [].

PUT передаёт полное изменяемое состояние. PATCH {} сохраняет все поля, отсутствие body сохраняет его, body=null очищает, body="" записывает пустую строку. pinned=false применяется. Для PATCH null в title и pinned означает пропуск поля.

JSON читается обычным json.NewDecoder(r.Body).Decode(&input). Неизвестные поля игнорируются. POST и PUT требуют непустой title, пропущенные body и pinned получают значения nil и false.

Успешные ответы кодируются через json.NewEncoder(w).Encode(...), Content-Type — application/json. Ошибки отправляются через http.Error обычным текстом. Неверные ID, пагинация, пустой title и ошибка декодирования дают 400, отсутствующая заметка — 404, ошибка БД — 500 с текстом internal server error. Перед ответом 500 запиши исходную ошибку через log.Printf вместе с методом и путём запроса.

## Методы обработчика

В handler/handler.go объявлены модель Note, структура Handler с общим полем db *sql.DB, маршруты и заготовки шести методов. Создание обработчика — handler.New(db).

```go
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request)
func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request)
func (h *Handler) GetNotes(w http.ResponseWriter, r *http.Request)
func (h *Handler) ReplaceNote(w http.ResponseWriter, r *http.Request)
func (h *Handler) PatchNote(w http.ResponseWriter, r *http.Request)
func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request)
```

Для ручек разрешены только следующие импорты. Подключай те, которые используешь.

```go
"bytes"
"database/sql"
"encoding/json"
"errors"
"io"
"log"
"net/http"
"strconv"
"strings"
"time"
```

Каждый метод целиком обрабатывает свою ручку. Внутри него читай параметры и JSON, проверяй вход, выполняй SQL через h.db, обрабатывай ошибки и записывай HTTP-ответ. Передавай r.Context() в QueryRowContext, QueryContext и ExecContext. Значения SQL передавай параметрами.

ID доступен через r.PathValue("id"). Для одной записи используй QueryRowContext/Scan и sql.ErrNoRows. Для списка нужны QueryContext, Next/Scan, Close и Err. Для удаления проверь RowsAffected. Пустой список кодируется как [].

PATCH должен различать отсутствие body, null и пустую строку. Это можно сделать локальной структурой с json.RawMessage. Применяй изменение одним UPDATE с CASE/COALESCE, чтобы конкурентные изменения разных полей сохранялись.

## Порядок реализации

1. Заполни schema.sql.
2. Напиши CreateNote, GetNote и GetNotes.
3. Напиши ReplaceNote, PatchNote и DeleteNote.
4. Собери запуск в cmd/server.

В cmd/server подготовлены заготовки newHTTPServer(h) и runServer(ctx, listener, h, grace). По контракту сервер должен иметь положительные ReadHeaderTimeout, ReadTimeout, WriteTimeout и IdleTimeout. После отмены ctx прекращается приём соединений, активный запрос получает grace period. По истечении grace оставшиеся запросы отменяются через Close. runServer дожидается завершения обработчиков перед возвратом, затем main закрывает БД.

В main прочитай DATABASE_URL и HTTP_ADDR, открой общий sql.DB, вызови PingContext и настрой пул, создай handler.New(db), listener и signal.NotifyContext для SIGINT/SIGTERM. HTTP_ADDR по умолчанию 127.0.0.1:8080.

## Проверки

Тесты проверяют поведение через HTTP и фактическое состояние PostgreSQL. Они создают уникальные схемы и удаляют их при cleanup. Проверяются CRUD, пагинация, валидация, PATCH, конкурентные обновления и отмена заблокированного SQL-запроса. Отдельно проверяются схема и остановка сервера.

```sh
export TEST_DATABASE_URL='postgres://student@127.0.0.1:15432/go_practice?sslmode=disable'
go test -timeout=30s -v ./04_crud/d01_notes/...
go test -timeout=30s -race -count=1 ./04_crud/d01_notes/...
go vet ./04_crud/d01_notes/...
```

Без TEST_DATABASE_URL и DATABASE_URL PostgreSQL-тесты явно пропускаются. Для интеграции нужен учебный сервер на loopback:15432. Файл schema.sql с TODO даёт ожидаемый red проверки схемы, методы с not implemented дают red поведения.

Полный контракт можно проверить на готовой реализации через [инструкцию примера](../../examples/d01_notes/README.md).

## Локальный запуск

Подготовь локальный PostgreSQL на порту 15432 с ролью student. После реализации схемы создай отдельную базу и выполни команды из корня модуля.

```sh
createdb -h 127.0.0.1 -p 15432 -U student go_practice
export DATABASE_URL='postgres://student@127.0.0.1:15432/go_practice?sslmode=disable'
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f 04_crud/d01_notes/schema.sql
HTTP_ADDR=127.0.0.1:8080 go run ./04_crud/d01_notes/cmd/server
```

После запуска выполни клиентскую цепочку H02 create → get → replace → patch → delete → get404 и проверь строки в БД.
