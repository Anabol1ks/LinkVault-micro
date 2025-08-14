# Link Service (LinkVault Microservices)

Микросервис управления короткими ссылками и их аналитикой в экосистеме **LinkVault**. Отвечает за генерацию коротких ссылок, редиректы, хранение и агрегацию статистики переходов (для пользовательских ссылок), управление жизненным циклом (истечения, деактивация, очистка) и интеграцию с Auth Service для авторизации пользователей. Транспорт — gRPC (включены Health Check и reflection).

## Назначение и ключевая функциональность

Основные возможности:

1. Создание коротких ссылок (анонимно и для авторизованных пользователей)
2. Редирект по короткому коду → возврат оригинального URL (логика кликов выполняется на стороне клиента, сервис отдаёт URL)
3. Получение списка активных ссылок пользователя
4. Получение детальной информации по конкретной ссылке
5. «Удаление» (деактивация) пользовательской ссылки
6. Сбор и предоставление аналитики (кол-во кликов, уникальные IP, география, распределение по дням) — только для ссылок зарегистрированных пользователей
7. Получение списка кликов с деталями (IP, User-Agent, страна, регион, время)
8. Плановое обслуживание: автоматическая деактивация и удаление просроченных / деактивированных ссылок и связанных кликов

### Поведение для анонимных и авторизованных ссылок

| Сценарий | TTL по умолчанию | Можно задать `expire_after` | Сохраняются клики | Очистка |
|----------|------------------|-----------------------------|-------------------|---------|
| Анонимная ссылка | 7 дней | Нет (игнорируется) | Нет | После истечения удаляется планировщиком |
| Пользовательская (без срока) | Бессрочно | Можно указать | Да | Истёкшие деактивируются и далее удаляются, если остаются неактивными > 7 дней |
| Пользовательская (с `expire_after`) | Указанный срок | Да | Да | После истечения деактивируется → затем удаляется |

## Архитектура и компоненты

```
link-service/
  cmd/main.go                – точка входа (конфиг, БД, миграции, gRPC сервер, Auth client, планировщик)
  config/                    – загрузка переменных окружения
  internal/models/           – GORM модели (ShortLink, Click)
  internal/repository/       – доступ к БД (CRUD + аналитические запросы)
  internal/service/          – бизнес‑логика (создание ссылок, клики, статистика)
  internal/transport/grpc/   – gRPC методы LinkService + interceptors авторизации
  internal/maintenance/      – cron планировщик (ежедневная очистка 03:00)
  internal/storage/          – подключение и миграция PostgreSQL
  pkg/logger/                – инициализация zap‑логгера
  Dockerfile / docker-compose.yml / Makefile
```

### Поток создания и использования ссылки

1. (Опционально) Клиент аутентифицируется через Auth Service и получает access‑токен.
2. Метод `CreateShortLink`:
   - Если запрос без токена → создаётся анонимная ссылка с TTL = 7 дней.
   - Если с токеном → ссылка привязана к пользователю; TTL не задан (бессрочно), либо ограничен параметром `expire_after`.
3. Метод `RedirectLink` возвращает оригинальный URL по коду (клик не сохраняется для анонимных ссылок).
4. Для пользовательских ссылок при редиректе в фоне фиксируется клик (IP, User-Agent, страна, регион).
5. Методы статистики доступны только владельцу ссылки.

### Очистка и деактивация (maintenance)

Ежедневно в 03:00 (и один раз при старте) cron выполняет:
- Деактивацию истёкших ссылок (анонимных и пользовательских)
- Удаление:
  - Истёкших анонимных ссылок
  - Анонимных деактивированных истёкших ссылок
  - Пользовательских истёкших ссылок, которые были деактивированы и старше 7 дней
  - При удалении удаляются связанные клики

## Переменные окружения

Используйте файл `.env` на основе шаблона `.env.example`:

| Переменная | Обяз. | Назначение | Пример | Примечание |
|------------|-------|-----------|--------|-----------|
| DB_HOST | yes | Хост PostgreSQL | link-db | В docker-compose — сервис `link-db` |
| DB_PORT | yes | Порт PostgreSQL | 5432 |  |
| DB_USER | yes | Пользователь БД | postgres |  |
| DB_PASSWORD | yes | Пароль БД | linkv12341 |  |
| DB_NAME | yes | Имя БД | link-db |  |
| DB_SSLMODE | yes | SSL режим | disable | Для локальной разработки |
| APP_PORT | yes | gRPC порт | :8082 | Формат `:порт` |
| DOMAIN | yes | Базовый домен для генерации short URL | http://localhost:8082 | Используется для ответа `ShortUrl` |
| ENV | yes | Окружение (`development` / `production`) | development | Меняет режим логгера |
| KAFKA_BROKERS | yes | Список брокеров Kafka | host.docker.internal:9092 | Подготовлено для будущих событий (пока не используется) |
| KAFKA_TOPIC_EMAIL | yes | Топик email событий | emails.send | Зарезервировано |
| AUTH_SERVICE_ADDR | yes | Адрес Auth Service (gRPC) | host.docker.internal:8081 | Для валидации access‑токенов |

Пример `.env`:

```env
DB_HOST=link-db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=linkv12341
DB_NAME=link-db
DB_SSLMODE=disable

APP_PORT=:8082
DOMAIN=http://localhost:8082

ENV=development

KAFKA_BROKERS=host.docker.internal:9092
KAFKA_TOPIC_EMAIL=emails.send
AUTH_SERVICE_ADDR=host.docker.internal:8081
```

## Запуск

### Предварительные требования
- Go 1.24+
- Docker / Docker Compose (для контейнерного запуска)
- PostgreSQL 15+ (или через Docker)
- Запущенный Auth Service (для защищённых методов)

### Вариант 1: Локально
1. Создайте `.env`.
2. Убедитесь, что PostgreSQL доступен и соответствующие значения заданы.
3. При необходимости запустите Auth Service и установите `AUTH_SERVICE_ADDR`.
4. Выполните:

```bash
go mod download
go run cmd/main.go
```

gRPC сервер слушает адрес из `APP_PORT` (по умолчанию `:8082`).

### Вариант 2: Docker Compose
В каталоге сервиса:

```bash
docker compose up -d --build
```

Или с Makefile:

```bash
make doc
```

Сервис поднимет контейнеры `link-db` и `link-service`.

### Makefile цели

| Цель | Описание |
|------|----------|
| `make run` | Локальный запуск (без Docker) |
| `make doc` | Поднятие через docker-compose (detached) |

## gRPC API

Протофайлы: репозиторий [linkvault-proto](https://github.com/Anabol1ks/linkvault-proto) (пакет `link/v1`). Ниже — обзор реализованных методов (поля упрощены по фактическому использованию в коде).

| Метод | Запрос (основные поля) | Ответ (основные поля) | Авторизация | Назначение |
|-------|------------------------|------------------------|-------------|-----------|
| CreateShortLink | `original_url`, `expire_after?` (duration строка, напр. `24h`) | `ShortLinkResponse { id, short_url, original_url, short_code, user_id?, expire_at, is_active }` | Опционально | Создание короткой ссылки |
| RedirectLink | `short_code` | `RedirectLinkResponse { original_url }` | Нет | Получение оригинального URL (для редиректа) |
| ListShortLinks | Empty | `ListShortLinksResponse { links[] }` | Bearer access | Список активных ссылок пользователя |
| GetShortLink | `id` | `ShortLinkResponse` | Bearer access | Детали конкретной ссылки |
| DeleteShortLink | `id` | `DeleteShortLinkResponse { message }` | Bearer access | Деактивация ссылки |
| GetLinkStats | `short_link_id` | `LinkStatsResponse { stats { total, unique_ip_count, unique_ips[], countries_count, countries[], daily_stats{date->count} } }` | Bearer access | Аггрегированная статистика кликов |
| GetLinkClicks | `short_link_id` | `GetLinkClicksResponse { clicks[] { id, ip, user_agent, clicked_at, country, region } }` | Bearer access | Сырые данные кликов |

### Особенности
- `expire_after` интерпретируется через `time.ParseDuration` (поддержка `s`, `m`, `h`); для бессрочной пользовательской ссылки поле пустое.
- Для анонимных ссылок статистика кликов не ведётся.
- Удаление — мягкое (деактивация). Физическое удаление происходит планировщиком.

### Примеры вызовов (grpcurl)

```bash
# Создание анонимной ссылки
grpcurl -plaintext -d '{"original_url":"https://example.com"}' localhost:8082 link.v1.LinkService/CreateShortLink

# Создание пользовательской ссылки с access токеном и кастомным сроком жизни (24 часа)
grpcurl -plaintext \
  -H 'authorization: Bearer ACCESS_TOKEN' \
  -d '{"original_url":"https://example.com","expire_after":"24h"}' \
  localhost:8082 link.v1.LinkService/CreateShortLink

# Список ссылок пользователя
grpcurl -plaintext -H 'authorization: Bearer ACCESS_TOKEN' -d '{}' localhost:8082 link.v1.LinkService/ListShortLinks

# Статистика ссылки
grpcurl -plaintext -H 'authorization: Bearer ACCESS_TOKEN' -d '{"short_link_id":"LINK_ID"}' localhost:8082 link.v1.LinkService/GetLinkStats
```

## Логирование
Используется `zap`. В режиме `development` включены человеко‑читаемые цветные логи; при завершении вызывается `logger.Sync()`.

## База данных и миграции
GORM `AutoMigrate` запускается на старте (`ShortLink`, `Click`). В продакшене рекомендуется перейти на управляемые миграции (например, `golang-migrate` / `atlas`).

## Планировщик (maintenance)
Cron (robfig/cron) выполняет ежедневные задачи (03:00) + однократная очистка при запуске:
- Деактивация истёкших ссылок
- Удаление старых анонимных / деактивированных ссылок и связанных кликов

## Взаимодействие с Auth Service
Все методы из списка `authRequiredMethods` в interceptor требуют валидного Bearer access‑токена, который проверяется удалённо через `ValidateAccessToken` (gRPC вызов Auth Service). Для `CreateShortLink` авторизация опциональна — при наличии токена ссылка привязывается к пользователю, иначе создаётся анонимная.

## Статистика и аналитика
Метод `GetLinkStats` возвращает агрегированные показатели, а `GetLinkClicks` — детальный список кликов (сортируется по времени по убыванию). Геоданные (страна, регион) получаются при создании клика через внешний HTTP сервис `ip-api.com` (best-effort; ошибки проигнорированы). **Примечание:** для приватности и производительности в продакшене стоит рассмотреть локальный GeoIP или прокси.

## Безопасность и рекомендации
- Храните секреты и доступы (пароли БД, адреса сервисов) вне Git (Vault / Kubernetes Secrets)
- Добавьте rate limiting / captcha на создание ссылок (анонимный спам)
- Рассмотрите защиту от утечек IP при геолокации (кеширование / отключаемый флаг)
- Добавьте аудит действий пользователя (создание/деактивация)

## Локальная разработка
- Для повторной генерации protobuf используйте репозиторий `linkvault-proto` — обновляйте версию модуля в `go.mod` при изменении.
- Тесты отсутствуют. Рекомендуется покрыть:
  1. Service слой (создание ссылок, вычисление TTL)
  2. Interceptor (обязательная / опциональная авторизация)
  3. Repository (статистика, выборки, очистка) — можно с тестовой PostgreSQL через Docker

## Возможные улучшения
- Перейти к versioned миграциям
- Добавить Prometheus метрики (создано ссылок, редиректы, время ответа)
- Кеширование коротких ссылок (Redis) для ускорения редиректов
- Поддержка пользовательских alias (кастомный shortCode)
- Ограничения на количество ссылок / кликов для тарифов
- Асинхронная запись кликов (batch + очередь)
- OpenTelemetry трейсинг межсервисных вызовов

## Лицензия
### [MIT](https://github.com/Anabol1ks/LinkVault-micro/blob/master/LICENSE)
