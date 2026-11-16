# CRUD заметок

Полная реализация [D01](../../04_crud/d01_notes/README.md) с net/http, encoding/json и database/sql через pgx/stdlib. Шесть обработчиков выполняют чтение запроса, валидацию, SQL и отправку ответа внутри отдельных методов.

## Устройство

- [handler/handler.go](handler/handler.go) содержит CreateNote, GetNote, GetNotes, ReplaceNote, PatchNote и DeleteNote с сигнатурами `func(w http.ResponseWriter, r *http.Request)`.
- [schema.sql](schema.sql) создаёт таблицу и ограничения.
- [cmd/server/main.go](cmd/server/main.go) настраивает БД, HTTP-сервер и graceful shutdown.
- [check.py](check.py) проверяет пример контрактными тестами учебного D01 в отдельной временной копии.

Обработчик создаётся через handler.New(db). Значения SQL передаются параметрами, запросы используют r.Context(). Успешные ответы кодируются через json.NewEncoder, ошибки отправляются через http.Error. Ошибки БД записываются в журнал.

## Подготовка PostgreSQL

Понадобятся PostgreSQL, утилиты initdb, pg_ctl, createdb и psql, а также Python 3 для проверки. Команды выполняются из корня go-practice. Для отдельного учебного кластера на свободном локальном порту 15432 выполни следующие команды один раз.

```sh
mkdir -p .runtime
initdb -D .runtime/notes-postgres -U student --auth-local=trust --auth-host=trust
pg_ctl -D .runtime/notes-postgres -l .runtime/notes-postgres.log \
  -o '-h 127.0.0.1 -p 15432' -w start
createdb -h 127.0.0.1 -p 15432 -U student go_practice_reference
export DATABASE_URL='postgres://student@127.0.0.1:15432/go_practice_reference?sslmode=disable'
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f examples/d01_notes/schema.sql
```

Если локальный учебный PostgreSQL с ролью student уже работает на этом порту, начни с создания новой базы. Конфигурация trust предназначена для этого изолированного кластера, слушающего 127.0.0.1.

## Запуск

```sh
export DATABASE_URL='postgres://student@127.0.0.1:15432/go_practice_reference?sslmode=disable'
HTTP_ADDR=127.0.0.1:8081 go run ./examples/d01_notes/cmd/server
```

В другом терминале создай заметку.

```sh
curl -i http://127.0.0.1:8081/notes \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Go","body":"HTTP","pinned":false}'
```

Используй id из ответа для остальных операций.

| Запрос | Результат |
| --- | --- |
| GET /notes/{id} | Заметка |
| GET /notes?limit=20&offset=0 | Страница заметок |
| PUT /notes/{id} | Полная замена изменяемых полей |
| PATCH /notes/{id} | Изменение переданных полей |
| DELETE /notes/{id} | Удаление с ответом 204 |

PATCH различает отсутствие поля, null, пустую строку и false. Полный HTTP-контракт и примеры JSON находятся в [условии D01](../../04_crud/d01_notes/README.md).

## Проверка

```sh
export TEST_DATABASE_URL='postgres://student@127.0.0.1:15432/go_practice_reference?sslmode=disable'
python3 examples/d01_notes/check.py
```

Скрипт создаёт временную копию D01, подставляет реализацию примера, запускает тесты с race detector и go vet. Проверки используют настоящую PostgreSQL и автоматически удаляемые схемы. Проверяются CRUD, PATCH, конкурентные обновления, отмена SQL, конечные HTTP-таймауты и остановка сервера.

## Проверено

16 ноября 2026 года check.py прошёл на новом временном кластере PostgreSQL 18.6 с ролью student и базой go_practice_reference на 127.0.0.1:15432. Тесты cmd/server, handler и internal/testdb выполнены с `-race`, `go vet` завершился успешно. После проверки временный кластер остановлен и удалён.

## Завершение работы

Ctrl+C или SIGTERM останавливает HTTP-сервер. Активным запросам выделяется 5 секунд, затем Server.Close отменяет оставшиеся запросы. После завершения обработчиков закрывается sql.DB.

Для остановки созданного выше учебного PostgreSQL выполни команду из корня модуля.

```sh
pg_ctl -D .runtime/notes-postgres -m fast -w stop
```

HTTP_ADDR по умолчанию равен 127.0.0.1:8081. ReadHeaderTimeout составляет 3 секунды, ReadTimeout и WriteTimeout — 15 секунд, IdleTimeout — 1 минуту. Пул БД допускает 6 соединений и 3 idle, lifetime составляет 5 минут. PingContext ограничен 3 секундами.
