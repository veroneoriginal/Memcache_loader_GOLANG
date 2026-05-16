# ── Этап 1: сборка
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache protobuf protobuf-dev && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

WORKDIR /build

# Кешируем зависимости отдельным слоем
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальное
COPY appsinstalled.proto ./
COPY src/ ./src/

# Генерируем protobuf в src/appsinstalled/
RUN mkdir -p src/appsinstalled && \
    protoc \
      --go_out=src/appsinstalled \
      --go_opt=paths=source_relative \
      appsinstalled.proto

# Компилируем — указываем путь к пакету main
RUN CGO_ENABLED=0 GOOS=linux go build -o memc_loader ./src

# ── Этап 2: финальный образ
FROM scratch

WORKDIR /app
COPY --from=builder /build/memc_loader .
VOLUME ["/app/data"]

ENTRYPOINT ["./memc_loader"]
CMD ["--pattern=/app/data/*.tsv.gz"]