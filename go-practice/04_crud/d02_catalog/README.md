# GO-D02 · Каталог — связи, JOIN и UPSERT

Частичная реализация. В handler/handler.go сохранён CreateCategory с UPSERT, маршрут зарегистрирован как POST /notes. Сервер и тесты вызывают отсутствующий handler.New, поэтому пакет не собирается. Остальные маршруты и модели полного контракта предстоит добавить.

Работающий сервис с прямыми HTTP-обработчиками находится в [примере CRUD заметок](../../examples/d01_notes/README.md).

## Что писать

Приведи конструктор к ожидаемому тестами New(db), зарегистрируй маршруты через mux.HandleFunc и реализуй методы из контракта в том же файле. Общий доступ к БД хранится в Handler.db.

Каждая ручка принимает w http.ResponseWriter и r *http.Request. Внутри неё читай параметры и JSON, выполняй SQL через h.db с r.Context(), отправляй ответ. Ошибки — через http.Error, логирование — через log.Printf.

## Запуск

После исправления сборки команды выполняются из корня go-practice.

```sh
go run ./04_crud/d02_catalog/cmd/server
```

Сервер по умолчанию ожидает базу go_interview_catalog на 127.0.0.1:15432 с пользователем student. HTTP-сервер слушает 127.0.0.1:8084. DATABASE_URL и HTTP_ADDR позволяют задать другие значения. В main.go настроены пул соединений, PingContext, таймауты HTTP и остановка по Ctrl+C/SIGTERM.

## Таблицы

Схема находится в schema.sql. Для приложения создай отдельную базу и примени этот файл. Каждая интеграционная проверка применяет его в отдельной автоматически удаляемой схеме PostgreSQL.

| Таблица | Поля и ограничения |
|---|---|
| categories | id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL UNIQUE |
| products | id BIGSERIAL PRIMARY KEY, sku TEXT NOT NULL UNIQUE, category_id BIGINT NOT NULL REFERENCES categories(id), name TEXT NOT NULL, price_cents BIGINT NOT NULL CHECK >= 0, stock BIGINT NOT NULL CHECK >= 0, description TEXT NULL |

Удаление используемой категории ограничено внешним ключом. Цена хранится целым количеством минимальных денежных единиц.

## JSON

Категория

```json
{"id":1,"name":"Книги"}
```

Товар

```json
{
  "id": 1,
  "sku": "GO-BOOK",
  "category_id": 1,
  "category_name": "Книги",
  "name": "Go",
  "price_cents": 120000,
  "stock": 5,
  "description": null
}
```

category_name получается через JOIN. Все ответы с товаром содержат это поле. В запросе создания или замены передаются sku, category_id, name, price_cents, stock и description.

## Ручки

| Запрос | Успех |
|---|---|
| POST /categories с {"name":"Книги"} | 201 и категория |
| GET /categories | 200 и массив категорий по id |
| GET /categories/{id} | 200 и категория |
| PUT /categories/{id} с {"name":"Учебники"} | 200 и категория |
| DELETE /categories/{id} | 204 |
| POST /products | 201 и товар |
| GET /products/{id} | 200 и товар с category_name |
| GET /products | 200 и массив товаров с category_name |
| PUT /products/{id} | 200 и полностью заменённые изменяемые поля |
| PATCH /products/{id} | 200 и изменённый товар |
| DELETE /products/{id} | 204 |
| PUT /products/by-sku/{sku} | 200 и товар при создании и обновлении |

Пустые списки возвращаются как []. У GET /products есть category_id, limit, offset и sort. По умолчанию limit=20, offset=0, sort=id. Допустимы limit от 1 до 100 и offset >= 0. sort принимает id, name или price_cents, порядок возрастающий. При равенстве значений сортируй по id.

PATCH изменяет переданные поля. description отсутствует — сохранить, null — очистить, пустая строка — записать пустую строку. price_cents=0 и stock=0 применяются.

## UPSERT

У PUT /products/by-sku/{sku} SKU берётся из пути. В JSON передаются category_id, name, price_cents, stock и description. Используй INSERT ... ON CONFLICT (sku) DO UPDATE ... RETURNING.

Один SKU соответствует одной строке. Повторный запрос сохраняет id и заменяет изменяемые поля. Два конкурентных запроса оставляют одну строку, её состояние целиком соответствует одному из запросов.

## Ошибки

Ошибки отправляй через http.Error.

- 400 при неправильном ID, пустом name или sku, отрицательных price_cents или stock, неверной пагинации или sort.
- 404 при обращении к отсутствующему ресурсу.
- 409 при повторе UNIQUE-значения или нарушении FK, включая удаление используемой категории и ссылку на отсутствующую категорию.
- 500 при внутренней ошибке БД.

При отказе существующие данные сохраняются. Значения передавай в SQL параметрами, sort выбирай из разрешённого перечня. SQL выполняется с r.Context().

## Проверки

```sh
export TEST_DATABASE_URL='postgres://student@127.0.0.1:15432/go_interview_catalog?sslmode=disable'
go test -timeout=30s -v ./04_crud/d02_catalog/handler
go test -timeout=30s -race -count=1 ./04_crud/d02_catalog/handler
```

Тесты проверяют HTTP-ответы и фактические строки PostgreSQL, CRUD, JOIN, фильтры, сортировку, UNIQUE/FK, PATCH и последовательный/конкурентный UPSERT. Общая подготовка БД находится в 04_crud/internal/testdb.

## Продолжение работы

Сначала исправь контракт конструктора и регистрации маршрутов. После восстановления сборки существующие тесты проверяют полный HTTP-контракт и ограничения SQL-схемы. Для интеграционных тестов создай отдельную базу go_interview_catalog на локальном PostgreSQL с портом 15432.
