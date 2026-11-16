# ЗАДАНИЕ: multi-stage Dockerfile для этого сервиса.
# Требования:
#  - стадия сборки на golang:1.26
#  - порядок слоёв: кеш зависимостей НЕ инвалидируется правкой main.go
#  - финальный образ: scratch (или distroless, если захочешь exec)
#  - бинарник статический (вспомни флаг!)
#  - ENTRYPOINT в exec-форме (сигналы должны доходить — там graceful shutdown)
#
# Сборка и запуск:
#   docker build -t day13 .
#   docker run --rm -p 8080:8080 day13
#   curl localhost:8080          ← в другой вкладке
#   docker stop <ctrl+c / container>  ← смотри в логи: graceful постройка отработает?
#
# TODO: пиши ниже


FROM golang:1.26 as builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM scratch
COPY --from=builder /server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/server"]
