# Auth Service

Сервис авторизации и регистрации пользователей для URL Shortener.

## Описание

Этот сервис предоставляет функциональность:
- Регистрация новых пользователей
- Авторизация существующих пользователей
- Генерация JWT токенов
- Валидация токенов

## API Endpoints

- `POST /api/auth/register` - регистрация нового пользователя
- `POST /api/auth/login` - авторизация пользователя
- `GET /api/auth/profile` - защищенный эндпоинт для проверки токена

## Конфигурация

| Переменная окружения | Описание | Значение по умолчанию |
|---------------------|----------|----------------------|
| SERVER_ADDRESS | Адрес сервера | :8082 |
| DATABASE_DSN | Строка подключения к БД | "" |
| JWT_SECRET | Секретный ключ для JWT | "insecure-default-change-me" |
| REFRESH_TOKEN_TTL | Срок жизни refresh-токена (в часах) | 720 |

## Запуск

### Локально
### Docker Compose (внешний PostgreSQL)

1. Создайте `.env` с переменными окружения (или экспортируйте их в окружение):

```
DB_DSN=postgres://user:password@host:5432/dbname?sslmode=disable
JWT_SECRET=change-me
RUN_MIGRATIONS=true
```

2. Соберите и запустите:

```
docker compose up --build -d
```

3. Healthcheck: `GET http://localhost:8081/healthz`

```bash
go run cmd/auth/main.go
```

### Docker
```bash
docker build -t auth-service .
docker run -p 8082:8082 auth-service
```

## Использование

### Регистрация
```bash
curl -X POST http://localhost:8082/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"login":"user1","password":"password123"}'
```

### Вход
```bash
curl -X POST http://localhost:8082/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"user1","password":"password123"}'
```

### Проверка токена
```bash
curl -X GET http://localhost:8082/api/auth/profile \
  -H "Authorization: Bearer <token>"
```

## Таблицы

- `users` — логины/хеши паролей/идентификаторы
- `profiles` — email, дата создания
- `refresh_tokens` — токен, user_id, expires_at, revoked