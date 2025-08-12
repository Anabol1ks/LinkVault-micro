# Auth Service (LinkVault Microservices)

Микросервис аутентификации и управления пользователями для экосистемы **LinkVault**. Отвечает за регистрацию, выдачу и валидацию JWT, управление refresh‑токенами, подтверждение email, восстановление пароля и интеграцию с email-уведомлениями через Kafka. Реализован как gRPC‑служба.

## Назначение и возможности

Основные функции:

1. Регистрация пользователя (c отложенной верификацией email)
2. Аутентификация (Login) и выдача пары Access / Refresh токенов
3. Обновление (Refresh) токенов с ревокацией использованного refresh
4. Получение профиля текущего пользователя
5. Logout (ревокация всех активных refresh‑токенов пользователя)
6. Подтверждение email + повторная отправка письма верификации
7. Запрос и подтверждение сброса пароля
8. Валидация access‑токена (для межсервисного взаимодействия)
9. Автоматическая очистка просроченных токенов (refresh / email verification / password reset)
10. Отправка email‑событий в Kafka (верификация, сброс пароля)

Транспорт: gRPC (поддерживается стандартный gRPC Health Check + reflection).

## Архитектура и компоненты

```
auth-service/
  cmd/main.go                – точка входа (инициализация конфигурации, БД, Kafka, gRPC)
  config/                    – загрузка переменных окружения
  internal/models/           – модели GORM (User, RefreshToken, EmailVerificationToken, PasswordResetToken)
  internal/repository/       – слой доступа к БД (GORM)
  internal/service/          – бизнес-логика (регистрация, login, refresh, верификация и т.д.)
  internal/jwt/              – генерация и парсинг JWT (access/refresh)
  internal/producer/         – Kafka producer для email-событий
  internal/maintenance/      – планировщик cron (очистка просроченных токенов в 03:00)
  internal/transport/grpc/   – gRPC сервер, методы AuthService, interceptor аутентификации
  internal/storage/          – подключение и миграции PostgreSQL (AutoMigrate)
  pkg/logger/                – инициализация zap-логгера
  Dockerfile / docker-compose.yml
  .env (локально; пример ниже)
```

### Поток авторизации

1. Пользователь регистрируется → создаётся запись User → генерируется EmailVerificationToken → отправляется сообщение в Kafka.
2. При Login валидируются email/пароль → создаются access и refresh токены → refresh сохраняется с JTI в БД.
3. Метод Refresh проверяет refresh JWT + наличие и валидность записи в БД → ревокация старого → выпуск новой пары.
4. Interceptor проверяет наличие Bearer access‑токена для защищённых методов (профиль, logout, resend email).
5. Планировщик ежедневно очищает просроченные / использованные токены.

### JWT

Два секрета (ACCESS_SECRET, REFRESH_SECRET). Сроки хранения задаются как duration (например `15m`, `24h`, `7d`). Refresh‑токены хранятся в БД (с JTI, временем истечения и признаками revoked), что позволяет массовую ревокацию на Logout.

### Kafka интеграция

Producer отправляет JSON:

```jsonc
{
  "to": "user@example.com",
  "subject": "Подтвердите email",
  "template": "verify_email",
  "data": { "UserName": "Alice", "ConfirmURL": "https://app/confirm?token=..." }
}
```

Конкретный consumer реализован в `notification-service` (другой микросервис). Топик задаётся переменной `KAFKA_TOPIC_EMAIL`.

## Переменные окружения

| Переменная | Обязат. | Назначение | Пример | Примечание |
|-----------|---------|-----------|--------|------------|
| DB_HOST | yes | Хост PostgreSQL | localhost |  |
| DB_PORT | yes | Порт PostgreSQL | 5432 |  |
| DB_USER | yes | Пользователь БД | postgres |  |
| DB_PASSWORD | yes | Пароль БД | linkv12341 |  |
| DB_NAME | yes | Имя БД | auth-db |  |
| DB_SSLMODE | yes | Режим SSL | disable | Для локальной разработки `disable` |
| APP_PORT | yes | Адрес прослушивания gRPC | :8081 | Формат допускает префикс `:` |
| ENV | yes | Окружение (`development` / `production`) | development | Влияет на формат логов |
| ACCESS_SECRET | yes | Секрет для подписи access JWT | ... | Должен быть достаточно длинным |
| ACCESS_EXP | yes | TTL access токена | 15m | Поддержка суффиксов `s,m,h` |
| REFRESH_SECRET | yes | Секрет для подписи refresh JWT | ... |  |
| REFRESH_EXP | yes | TTL refresh токена | 7d | Поддержка суффикса `d` (дни) |
| KAFKA_BROKERS | yes | Комма-разделённый список брокеров | host.docker.internal:9092 | Пример для Docker Desktop |
| KAFKA_TOPIC_EMAIL | yes | Топик email-событий | emails.send |  |

Пример `.env` (не коммить в репозиторий):

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=linkv12341
DB_NAME=auth-db
DB_SSLMODE=disable

APP_PORT=:8081

ENV=development

ACCESS_SECRET=change-me-access-secret
ACCESS_EXP=15m
REFRESH_SECRET=change-me-refresh-secret
REFRESH_EXP=7d

KAFKA_BROKERS=host.docker.internal:9092
KAFKA_TOPIC_EMAIL=emails.send
```

## Запуск

### Предварительные требования

- Go 1.24+
- Docker / Docker Compose (для контейнерного запуска)
- PostgreSQL 15+ (или через Compose)
- Kafka кластер (локально можно использовать single‑broker)

### Вариант 1: Локально (без Docker)

1. Создать/сконфигурировать `.env`.
2. Убедиться, что PostgreSQL и Kafka доступны по указанным адресам.
3. Выполнить:

```bash
go mod download
go run cmd/main.go
```

gRPC сервер слушает порт из `APP_PORT` (по умолчанию `:8081`).

### Вариант 2: Docker Compose
Замените в `.env` `DB_HOST=localhost` на `DB_HOST=auth-db`.
В репозитории есть `docker-compose.yml`, поднимающий PostgreSQL и сервис:

```bash
docker compose up -d --build
```

Kafka по умолчанию не включён. Можно добавить сервис Kafka, например:

```yaml
  services:
  kafka:
    image: apache/kafka:4.0.0
    container_name: kafka
    environment:
      - KAFKA_PROCESS_ROLES=broker,controller
      - KAFKA_NODE_ID=1
      - KAFKA_LISTENERS=INTERNAL://0.0.0.0:29092,EXTERNAL://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      - KAFKA_ADVERTISED_LISTENERS=INTERNAL://kafka:29092,EXTERNAL://host.docker.internal:9092
      - KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT,CONTROLLER:PLAINTEXT
      - KAFKA_INTER_BROKER_LISTENER_NAME=INTERNAL
      - KAFKA_CONTROLLER_QUORUM_VOTERS=1@kafka:9093
      - KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
      - KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1
      - KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1
      - KAFKA_AUTO_CREATE_TOPICS_ENABLE=true
      - KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER
    ports:
      - "9092:9092"   # внешний доступ (EXTERNAL)
      - "29092:29092" # при необходимости тестов из других контейнеров можно тоже использовать
    healthcheck:
      test: ["CMD", "kafka-broker-api-versions", "--bootstrap-server", "localhost:9092"]
      interval: 10s
      timeout: 5s
      retries: 5
```

И скорректировать `KAFKA_BROKERS=kafka:9092` (если запущен в отдельном контейнере, то `KAFKA_BROKERS=host.docker.internal:9092`).

### Makefile цели

| Цель | Описание |
|------|----------|
| `make run` | Локальный запуск (Go) |
| `make doc` | Поднятие через docker-compose (detached) |

## gRPC API

Протофайлы находятся в репозитории: [github.com/Anabol1ks/linkvault-proto](https://github.com/Anabol1ks/linkvault-proto) (пакет `auth/v1`).

Методы сервиса `AuthService`:

| Метод | Запрос | Ответ | Авторизация | Назначение |
|-------|--------|-------|-------------|-----------|
| Register | RegisterRequest { name, email, password } | RegisterResponse { id, name, email } | Нет | Создание пользователя + письмо верификации |
| Login | LoginRequest { email, password } | TokenPair { access_token, refresh_token } | Нет | Аутентификация |
| Refresh | RefreshRequest { refresh_token } | TokenPair | Нет | Выпуск новой пары токенов |
| GetProfile | GetProfileRequest (пусто) | UserProfile { id, name, email } | Bearer access | Профиль текущего пользователя |
| Logout | LogoutRequest (пусто) | Empty | Bearer access | Ревокация всех refresh пользовательских токенов |
| ValidateAccessToken | ValidateAccessTokenRequest { access_token } | ValidateAccessTokenResponse { user_id, valid } | Нет | Проверка валидности (для других микросервисов) |
| VerifyEmail | VerifyEmailRequest { token } | Empty | Нет | Подтверждение email |
| ResendVerificationEmail | ResendVerificationEmailRequest (пусто) | Empty | Bearer access | Повторная отправка письма |
| RequestPasswordReset | RequestPasswordResetRequest { email } | Empty | Нет | Инициировать сброс пароля |
| ConfirmPasswordReset | ConfirmPasswordResetRequest { token, new_password } | Empty | Нет | Подтвердить сброс пароля |

### Пример вызовов (grpcurl)

```bash
# Регистрация
grpcurl -plaintext -d '{"name":"Alice","email":"alice@example.com","password":"Secret123!"}' localhost:8081 auth.v1.AuthService/Register

# Логин (получение токенов)
grpcurl -plaintext -d '{"email":"alice@example.com","password":"Secret123!"}' localhost:8081 auth.v1.AuthService/Login

# Профиль (замените ACCESS_TOKEN)
grpcurl -plaintext -H 'authorization: Bearer ACCESS_TOKEN' -d '{}' localhost:8081 auth.v1.AuthService/GetProfile

# Refresh
grpcurl -plaintext -d '{"refresh_token":"REFRESH_TOKEN"}' localhost:8081 auth.v1.AuthService/Refresh
```

### Ошибки и статусы

Используются стандартные `gRPC codes`:
- `InvalidArgument` – валидация входных данных (Validate() из proto)
- `AlreadyExists` – пользователь уже существует / email уже подтверждён
- `NotFound` – пользователь не найден
- `Unauthenticated` – неверные креды / токен / отсутствует авторизация
- `Internal` – прочие ошибки

## Планировщик (maintenance)

Cron-задача: ежедневно в 03:00 серверного времени:
- Удаляет просроченные или revoked refresh токены
- Удаляет просроченные/использованные email verification токены
- Удаляет просроченные/использованные password reset токены

Очистка также запускается один раз при старте.

## Логирование

Используется `zap`. В режиме `development` – цветные уровни, подробные поля. При завершении приложения вызывается `logger.Sync()`.

## Миграции

GORM `AutoMigrate` применяется на старте ко всем моделям. В продакшене рекомендуется перейти на управляемые миграции (go-migrate / atlas) для контроля схемы.

## Безопасность и рекомендации

- Храните секреты (ACCESS_SECRET / REFRESH_SECRET) вне Git (Vault / Kubernetes Secrets)
- Минимизируйте TTL access (короткий) и оценивайте необходимость длинного refresh
- Добавьте rate limiting / captcha на Register и Login (отсутствует в текущей версии)
- Рассмотрите blacklisting IP при множественных неудачных логинах

## Локальная разработка

Перегенерация protobuf (пример):

```bash
protoc --go_out=. --go-grpc_out=. proto/auth/v1/auth.proto
```

Фактические файлы берутся из внешнего репозитория [linkvault-proto](https://github.com/Anabol1ks/linkvault-proto/tree/master/auth/v1); убедитесь в обновлении версии модуля в go.mod при изменениях.

## Тестирование (скоро будет)

Unit / интеграционные тесты в текущей версии отсутствуют. Рекомендуемые направления:
1. Сервисный слой (регистрация, refresh, сброс пароля)
2. Interceptor (валидация токена)
3. Repository (in-memory / dockerized PostgreSQL)

## Возможные дальнейшие улучшения

- Перейти от AutoMigrate к versioned миграциям
- Добавить метрики (Prometheus) и трейсинг (OpenTelemetry)
- Ввести ограничение частоты отправки писем (email verification / reset)
- Поддержка soft delete пользователей / audit trail
- Верификация пароля политиками (сложность, pwned check)

## Лицензия
### [MIT](https://github.com/Anabol1ks/LinkVault-micro/blob/master/LICENSE)


