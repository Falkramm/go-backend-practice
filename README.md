# Go — от задач к backend

Личная коллекция Даниила Матуса. Алгоритмы, горутины, сетевые запросы и HTTP-сервисы с PostgreSQL — всё, на чём я разбирался с Go и готовился к собеседованиям.

Собрано, чтобы однажды показать детям, как папа учился программировать на Go.

## Что открыть

| Тема | Где смотреть | Что внутри |
| --- | --- | --- |
| Алгоритмы | [Ранняя практика](early-practice/README.md) | Two Sum, sliding window, интервалы, скобки, списки, анаграммы и LRU |
| LeetCode | [Решения](leetcode/README.md) · [Программа задач](docs/leetcode-catalog.md) | 23 восстановленных Go-решения с вердиктом Accepted, включая LRU и LFU |
| Конкурентность | [C01–C11](go-practice/02_concurrency/README.md) | Каналы, fan-in, worker pool, pipeline, batching, mutex, rate limiter и token bucket |
| HTTP-клиенты | [H01](go-practice/03_http/h01_get/README.md) · [H02](go-practice/03_http/h02_json_methods/README.md) · [H03](go-practice/03_http/h03_timeouts/README.md) | Query-параметры, JSON, контекст, ограничения тела ответа и таймауты |
| Готовый CRUD | [Сервис заметок](go-practice/examples/d01_notes/README.md) | Шесть HTTP-обработчиков, SQL, PATCH, логирование и graceful shutdown |
| CRUD-практика | [Заметки](go-practice/04_crud/d01_notes/README.md) · [Каталог](go-practice/04_crud/d02_catalog/README.md) · [Заказы](go-practice/04_crud/d03_orders/README.md) | Учебные контракты, схемы, тесты и частичные реализации |
| Ранние подходы | [Практика по дням](early-practice/README.md) · [Interview-черновик](archive/interview/README.md) | Предыдущие версии упражнений и незавершённые эксперименты |

## Попробовать

Нужен Go 1.24 или новее. Для этих упражнений достаточно Go и доступа к загрузке зависимостей.

```sh
git clone https://github.com/Falkramm/go-backend-practice.git
cd go-backend-practice/go-practice
go test -race -count=1 -timeout=30s \
  ./01_errors_json/... \
  ./02_concurrency/c01_alternation \
  ./02_concurrency/c04_worker_pool \
  ./02_concurrency/c05_parallel_map \
  ./03_http/h01_get
```

Чтобы начать с алгоритмов, из корня репозитория выполни

```sh
cd early-practice/day2
go test -race -count=1 -timeout=15s ./...
```

У каждого раннего Go-проекта свой `go.mod`. У `go-practice` общий модуль. Команды запускаются из соответствующего модуля.

Для сервиса заметок понадобится PostgreSQL. Подготовка базы, запуск сервера и пример `curl` описаны рядом с реализацией. [Открыть инструкцию](go-practice/examples/d01_notes/README.md)

## Как читать эту коллекцию

В папках сохранены условия, реализации и тесты из разных этапов подготовки. Завершённые упражнения соседствуют с черновиками, в которых ещё встречаются `TODO`, заглушки и ошибки. Состояние каждого блока отмечено в его README.

При сборке 16 ноября 2026 года восстановлены 23 Accepted-решения LeetCode и проверены ранние модули day1–day9, ошибки и JSON, конкурентность C01–C10 и HTTP GET H01. Готовый CRUD заметок прошёл свои контракты с race detector и `go vet` на отдельной PostgreSQL 18.6. В остальных заданиях сохранились известные недоработки. Команды и результаты собраны отдельно. [Результаты проверок](docs/verification.md)

Учебные реализации и их ограничения сохранены. Оформление, навигация и примеры данных подготовлены для публикации. [Происхождение материалов](docs/sources.md)
