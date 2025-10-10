# Dockerfile для сервиса авторизации

# Используем официальный образ Go как базовый
FROM golang:1.25.2-alpine AS builder

# Установка переменных окружения
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum для загрузки зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

RUN go mod tidy && go mod download
# Сборка бинарного файла для сервиса авторизации
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o auth-service ./cmd/auth

# Используем минимальный образ Alpine для запуска приложения
FROM alpine:latest

# Установка рабочей директории
WORKDIR /root/

# Копирование бинарного файла из builder образа
COPY --from=builder /app/auth-service .
COPY ./migrations ./migrations

# Создание директории для файлового хранилища
RUN mkdir -p /tmp

# Экспонирование порта
EXPOSE 8081

# Add non-root user
RUN adduser -D -H appuser && chown -R appuser:appuser /root
USER appuser

# Команда запуска приложения
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8081/healthz || exit 1

CMD ["./auth-service"]
