# LinkVault Microservices

> Микросервисная платформа для аутентификации, сокращения ссылок и транзакционных email‑уведомлений (gRPC + Kafka + PostgreSQL).

---
## Содержание
- [LinkVault Microservices](#linkvault-microservices)
  - [Содержание](#содержание)
  - [Описание](#описание)
  - [Архитектура](#архитектура)
  - [Микросервисы](#микросервисы)
  - [gRPC Proto файлы](#grpc-proto-файлы)
  - [Пререквизиты](#пререквизиты)
  - [Переменные окружения](#переменные-окружения)
  - [Установка и запуск](#установка-и-запуск)
  - [Примеры использования / быстрые проверки](#примеры-использования--быстрые-проверки)
  - [Troubleshooting](#troubleshooting)
  - [Рекомендации по развитию](#рекомендации-по-развитию)
  - [Лицензия](#лицензия)
    - [MIT](#mit)

---
## Описание
LinkVault — учебно‑практический проект, демонстрирующий:
- построение набора независимых Go‑микросервисов (чистая логика + gRPC транспорт);
- асинхронное взаимодействие через Kafka (email события);
- надёжную работу с JWT (access / refresh) и токен‑ориентированную инвалидацию;
- хранение доменных данных в отдельных PostgreSQL базах (per‑service DB);
- инфраструктуру простого локального запуска через единый `docker-compose.yml`.

Проект подходит как демонстрация производственного стека для рекрутёров и база для дальнейшего масштабирования (API Gateway, observability, распределённый трейсинг, rate limiting, сервис аналитики и т.д.).

---
## Архитектура
Высокоуровневое взаимодействие:
```
[ Client / grpcurl / Future Gateway ]
        | (gRPC)                    
        v                           
 +----------------+        Kafka (topic: emails.send)        +---------------------+
 |  auth-service  |  --(produce email events)--------------> | notification-service |
 |  (JWT, Users)  |                                         | (SMTP sender)        |
 +----------------+                                         +---------------------+
        ^  \
        |   \
        |    \ (gRPC validate token)
        |     \
        v      \
 +----------------+   (DB per service)  +----------------+
 |  link-service   | <-----------------> |  PostgreSQL    |
 | (Short Links)   |                     +----------------+
 +----------------+
```
Ключевые каналы:
- gRPC — синхронные запросы клиента к `auth-service` и `link-service`.
- Kafka — асинхронные события (верификация email, сброс пароля) от `auth-service` к `notification-service`.
- Отдельные PostgreSQL экземпляры для каждого сервиса (изоляция схем и масштабирование).

---
## Микросервисы
| Сервис | Кратко | Транспорт | Хранилище | Коммуникация |
|--------|--------|-----------|-----------|--------------|
| `auth-service` | Регистрация, логин, refresh, email верификация, password reset | gRPC | PostgreSQL (`auth-db`) | Kafka producer → `emails.send` | 
| `link-service` | Сокращение ссылок, статистика кликов, управление TTL | gRPC | PostgreSQL (`link-db`) | gRPC запрос к Auth для валидации access токена |
| `notification-service` | Обработка email событий и отправка писем по шаблонам | (нет публичного API) | — | Kafka consumer (`emails.send`) + SMTP |

Подробности в README каждого сервиса:
- [`auth-service/README.md`](/auth-service/README.md)
- [`link-service/README.md`](/link-service/README.md)
- [`notification-service/README.md`](/notification-service/README.md)

---
## gRPC Proto файлы
Все protobuf спецификации (service / messages) вынесены в отдельный репозиторий: **[`linkvault-proto`](https://github.com/Anabol1ks/linkvault-proto)**.

Там находятся пакеты (например):
- [`auth/v1`](https://github.com/Anabol1ks/linkvault-proto/blob/master/auth/v1/auth.proto) — RPC для аутентификации, управления пользователями и валидации токенов
- [`link/v1`](https://github.com/Anabol1ks/linkvault-proto/blob/master/link/v1/link.proto) — операции с короткими ссылками и аналитикой
- будущие пакеты для уведомлений или gateway


---
## Пререквизиты
Обязательно:
- Docker Compose v2
- Git

Опционально (для локальной отладки вне контейнеров):
- 1.24+
- `grpcurl` для тестов gRPC
- `make`

---
## Переменные окружения
На корневом уровне используется файл `.env` (передаётся в compose) с кредами баз:
```
AUTH_DB_USER=postgres
AUTH_DB_PASSWORD=******
LINK_DB_USER=postgres
LINK_DB_PASSWORD=******
```
Остальные переменные определены в `.env` каждого сервиса (см. соответствующие README):
- `auth-service/.env`: DB параметры, JWT секреты (`ACCESS_SECRET`, `REFRESH_SECRET`), сроки (`ACCESS_EXP`, `REFRESH_EXP`), `KAFKA_BROKERS`, `KAFKA_TOPIC_EMAIL`.
- `link-service/.env`: DB, `DOMAIN`, `AUTH_SERVICE_ADDR`, Kafka резерв.
- `notification-service/.env`: SMTP (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`), Kafka consumer настройки.

Советы по секретам:
- Не коммить реальные секреты; используйте `.env.example` как шаблон.
- Для разработки допускается `host.docker.internal` (Windows/Mac). В Linux замените на `localhost` или сетевое имя контейнера.

---
## Установка и запуск
1. Клонировать репозиторий:
   ```bash
   git clone https://github.com/Anabol1ks/LinkVault.git
   cd LinkVault-micro
   ```
2. Создать корневой `.env` (на основе примера выше) и `.env` в каждом сервисе.
3. Проверить/отредактировать Kafka настройки (`KAFKA_BROKERS=host.docker.internal:9092`).
4. Запустить:
   ```bash
   docker compose up -d --build
   # или make shortcut
   make doc
   ```
5. Убедиться, что контейнеры здоровы:
   ```bash
   docker compose ps
   docker compose logs -f auth-service
   ```
6. (Опционально) Выполнить миграции вручную / проверить логи инициализации БД (в проекте код миграций выполняется при старте).

Остановка:
```bash
docker compose down -v
```

Пересборка после изменения Go-кода:
```bash
docker compose build auth-service
```

## Примеры использования / быстрые проверки
Ниже ориентировочные шаги (можно адаптировать под реальные proto / методы). Для gRPC вызовов используйте `grpcurl` (предполагая включён reflection — он активирован в сервисах).

1. Проверка Health (Auth):
   ```bash
   grpcurl -plaintext localhost:8081 grpc.health.v1.Health/Check
   ```
2. Регистрация пользователя (пример; имя метода условно):
   ```bash
   grpcurl -plaintext -d '{"email":"user@example.com","password":"Passw0rd!"}' localhost:8081 auth.AuthService/Register
   ```
3. Логин → получить токены:
   ```bash
   grpcurl -plaintext -d '{"email":"user@example.com","password":"Passw0rd!"}' localhost:8081 auth.AuthService/Login
   ```
4. Создание короткой ссылки (авторизованно):
   ```bash
   grpcurl -plaintext -H "authorization: Bearer <ACCESS_TOKEN>" -d '{"original_url":"https://golang.org"}' localhost:8082 link.LinkService/CreateShortLink
   ```
5. Получение оригинального URL по коду:
   ```bash
   grpcurl -plaintext -d '{"code":"abc123"}' localhost:8082 link.LinkService/Resolve
   ```
6. Тест Kafka → Email (эмуляция отправки события напрямую):
   ```bash
   # Используйте kafka-console-producer / kcat (пример формата):
   {"to":"user@example.com","template":"verify_email","data":{"UserName":"User","ConfirmURL":"https://app/confirm?token=..."}}
   ```

(Точные названия RPC уточняйте в proto / коде сервиса.)

---
## Troubleshooting
| Проблема | Причина | Решение |
|----------|---------|---------|
| Kafka недоступен из контейнера | Неверный listener / Windows host | Убедитесь в `KAFKA_ADVERTISED_LISTENERS` и `host.docker.internal`; для Linux замените на `PLAINTEXT://kafka:9092` и используйте сетевое имя. |
| Ошибка подключения к БД | Переменные / порт занят | Проверьте `.env`, что порты 5432 / 5434 свободны; при конфликте смените host‑порт. |
| JWT валидация падает | Просроченный токен / несоответствие секрета | Пересоздайте токены; убедитесь, что `ACCESS_SECRET` одинаков в конфигурации запуска и не изменён после выдачи. |
| Письма не отправляются | SMTP блокирует или неверный пароль приложения | Сгенерируйте пароль приложения, включите SSL порт, проверьте лог `notification-service`. |
| `grpcurl` не видит методы | Reflection выключен или порт недоступен | Проверьте логи сервиса; убедитесь, что проброшен нужный порт. |
| CodePage / UTF‑8 артефакты в логах Windows | Кодировка терминала | Выполните `chcp 65001` перед просмотром логов. |

---
## Рекомендации по развитию
Короткий roadmap для демонстрации инженерного мышления:
- API Gateway (REST + gRPC proxy) + централизованная аутентификация.
- Observability: Prometheus metrics, OpenTelemetry traces, структурированные кореллирующие request ID.
- Rate limiting / throttling (token bucket, Redis). 
- Circuit breaker / retry policy для межсервисных вызовов.
- Отдельный сервис аналитики ссылок (агрегация по батчам из Kafka вместо синхронных вставок).
- Dead Letter Topic (DLT) для некорректных email сообщений.
- Выделение схемы миграций (например, `golang-migrate` CLI и папка migrations). 
- Helm чарты / Kustomize для деплоя в Kubernetes.
- Vault / SOPS для секретов.
- Авто‑генерация gRPC Gateway + OpenAPI документации.

---
## Лицензия
### [MIT](https://github.com/Anabol1ks/LinkVault-micro/blob/master/LICENSE)


