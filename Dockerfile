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

# Сборка бинарного файла для сервиса авторизации
RUN go build -a -installsuffix cgo -o auth-service cmd/auth/main.go

# Используем минимальный образ Alpine для запуска приложения
FROM alpine:latest

# Установка рабочей директории
WORKDIR /root/

# Копирование бинарного файла из builder образа
COPY --from=builder /app/auth-service .

# Создание директории для файлового хранилища
RUN mkdir -p /tmp

# Экспонирование порта
EXPOSE 8081

# Команда запуска приложения
CMD ["./auth-service"]
