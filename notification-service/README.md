# Notification Service (LinkVault Microservices)

Микросервис отправки транзакционных email‑уведомлений экосистемы **LinkVault**. Обеспечивает асинхронную обработку событий (Kafka) и доставку писем через SMTP, используя шаблоны (plain + HTML) с подстановкой динамических данных.

## Назначение
`notification-service` принимает задачи на отправку писем из Kafka, рендерит соответствующие шаблоны и отправляет email пользователю. Сервис не предоставляет HTTP/gRPC API — взаимодействие идёт исключительно через очередь сообщений.

Типовые сценарии:
- Подтверждение email при регистрации
- Сброс пароля
- (Расширяемо) любые другие транзакционные уведомления

## Ключевая функциональность
- Kafka consumer (группа `notification-service`) с автоматическим чтением и коммитом оффсетов
- Приём JSON сообщений определённого формата (см. ниже)
- Рендер текстового (`.txt`) и HTML (`.html`) шаблонов (Go `text/template`/`html/template`)
- Отправка писем через SMTP (SSL) с альтернативными MIME частями (text/plain + text/html)
- Встраивание логотипа `icon.png` через `cid:logo` при наличии в HTML
- Централизованное логирование (`zap`) в dev / prod режимах
- Graceful shutdown при SIGINT / SIGTERM

## Структура проекта
```
notification-service/
  cmd/main.go              – точка входа (загрузка env, логгер, создание consumer)
  config/config.go          – загрузка и валидация переменных окружения
  internal/consumer/        – Kafka консьюмер (чтение и обработка email сообщений)
  internal/model/           – доменная модель `EmailNotification`
  internal/sender/          – логика рендера шаблонов и SMTP отправка
  internal/templates/       – HTML и TXT шаблоны + вложения (icon.png)
  pkg/logger/               – инициализация zap‑логгера
  Dockerfile / docker-compose.yml / Makefile / .env.example
```

## Переменные окружения
Используйте `.env` на основе шаблона `.env.example`.

| Переменная | Обязат. | Назначение | Пример | Примечание |
|-----------|---------|-----------|--------|-----------|
| SMTP_HOST | yes | Хост SMTP сервера | smtp.mail.ru |  |
| SMTP_PORT | yes | Порт SMTP (int) | 465 | SSL порт у Mail.ru / Yandex (пример) |
| SMTP_USER | yes | Логин / адрес почтового ящика | vashapochta@ru.ru |  |
| SMTP_PASSWORD | yes | Пароль приложения (не основной пароль) | ******** | Создаётся в настройках почты |
| SMTP_FROM | yes | Отображаемое имя / адрес отправителя | "LinkVault Notifications" | Можно в формате Имя <email> |
| ENV | no | Окружение (`development`/`production`) | development | Влияет на формат логов |
| TMPL_DIR | yes | Путь к директории шаблонов | internal/templates/ | Относительно корня запуска
| KAFKA_BROKERS | yes | Список брокеров через запятую | host.docker.internal:9092 | Для нескольких: `b1:9092,b2:9092` |
| KAFKA_GROUP_ID | yes | Group ID consumer | notification-service |  |
| KAFKA_TOPIC_EMAIL | yes | Kafka топик с заданиями | emails.send |  |

Пример `.env`:
```env
SMTP_HOST=smtp.mail.ru
SMTP_PORT=465
SMTP_USER=vashapochta@ru.ru
SMTP_PASSWORD=пароль_приложения
SMTP_FROM="LinkVault Notifications"

ENV=development

TMPL_DIR=internal/templates/

KAFKA_BROKERS=host.docker.internal:9092
KAFKA_GROUP_ID=notification-service
KAFKA_TOPIC_EMAIL=emails.send
```

## Формат Kafka сообщения
Сообщения читаются из топика `KAFKA_TOPIC_EMAIL` и должны быть валидным JSON со структурой:
```json
{
  "to": "user@example.com",
  "subject": "Подтвердите email",
  "template": "verify_email",
  "data": {
    "UserName": "Alice",
    "ConfirmURL": "https://app/confirm?token=..."
  }
}
```
| Поле | Тип | Обяз. | Описание |
|------|-----|-------|----------|
| to | string | Да | Email получателя |
| subject | string | Нет | Тема письма (рекомендуется указывать) |
| template | string | Да | Имя шаблона без расширения (`verify_email`, `reset_password`, …) |
| data | object | Нет | Пара ключ/значение для подстановки в шаблон |

Ошибки (повреждённый JSON, отсутствие обязательных полей) приводят к логированию и пропуску сообщения без ретрая на стороне сервиса.

### Примеры
Подтверждение email:
```json
{
  "to": "user@example.com",
  "subject": "Подтвердите email — LinkVault",
  "template": "verify_email",
  "data": {
    "UserName": "Alice",
    "ConfirmURL": "https://linkvault.app/confirm?token=..."
  }
}
```
Сброс пароля:
```json
{
  "to": "user@example.com",
  "subject": "Сброс пароля — LinkVault",
  "template": "reset_password",
  "data": {
    "UserName": "Alice",
    "ResetURL": "https://linkvault.app/reset?token=...",
    "ExpireMinutes": 30
  }
}
```

## Запуск
### Предварительные требования
- Go 1.24+
- Доступный Kafka брокер
- SMTP сервер / тестовый mock (например MailHog) (при тестировании)

### Локально (без Docker)
1. Скопируйте `.env.example` в `.env` и заполните значения.
2. Убедитесь, что Kafka и SMTP доступны.
3. Запустите:
```bash
make run
# или
go run ./cmd/main.go
```

### Docker / Docker Compose
В каталоге `notification-service`:
```bash
docker compose up -d --build
```
Логи:
```bash
docker compose logs -f notification-service
```
Остановка:
```bash
docker compose down
```

Healthcheck в `docker-compose.yml` проверяет существование процесса.

> Примечание: volume в текущем `docker-compose.yml` указывает на `./internal/sender/templates`, но реальные шаблоны лежат в `internal/templates`. Для live-редактирования можете заменить / добавить:
```yaml
    volumes:
      - ./internal/templates:/app/internal/templates:ro
```
И установить `TMPL_DIR=internal/templates/`.

### !Kafka по умолчанию не включён. Можно добавить сервис Kafka, например:

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
| `make run` | Локальный запуск через Go |
| `make doc` | Поднятие через docker-compose (detached) |

## Добавление нового email шаблона
1. Создайте пару файлов в `internal/templates/`: `<name>.html` и `<name>.txt`.
2. Используйте Go template синтаксис (`{{ .Key }}`).
3. (Опционально) добавьте `cid:logo` в HTML, чтобы встроить `icon.png` (не забудьте хранить файл в той же директории).
4. Отправьте тестовое Kafka сообщение с `"template": "<name>"` и нужными данными `data`.

## Отправка тестового сообщения
Используя консольный продьюсер (пример):
```bash
kafka-console-producer --broker-list localhost:9092 --topic emails.send
>{"to":"test@example.com","subject":"Test","template":"verify_email","data":{"UserName":"Test","ConfirmURL":"https://example.com/confirm?x=1"}}
```

## Логирование
- `ENV=development` включает человеко‑читаемые цветные логи
- В продакшене используется production конфиг zap
- События: старт консьюмера, успешные отправки, ошибки чтения/парсинга/SMTP

## Обработка ошибок
| Ситуация | Поведение |
|----------|-----------|
| Не хватает обязательной env | Panic на старте (fail fast) |
| Ошибка чтения сообщения | Логируется, цикл продолжается |
| Невалидный JSON / отсутствуют поля | Лог (warn/error), сообщение пропущено |
| Рендер шаблона | Лог (error), письмо не отправляется |
| Ошибка SMTP | Лог (error), без повторной попытки |

Ретраи целесообразно реализовывать на стороне продьюсера или с DLQ.

## Graceful Shutdown
При SIGINT / SIGTERM отменяется контекст, закрывается Kafka reader и даётся небольшое время завершить операцию (200ms). Это предотвращает частичную обработку сообщений.

## Рекомендации для продакшена
- Внешние ретраи / DLQ (dead-letter topic) для неотправленных писем
- Prometheus метрики (время рендера, успех/ошибка отправки, consumer lag)
- Circuit breaker / timeout для SMTP (использование стороннего email API провайдера в качестве fallback)
- Vault / Secret Manager для хранения секретов (не хранить в plain `.env`)
- Лимит исходящих писем (rate limiting) и алертинг при всплесках
- Валидация email адреса (формат / домен) до постановки в очередь (на стороне продьюсера)

## Тестирование (рекомендации)
Покрытие тестами отсутствует. Рекомендуется:
1. Юнит-тест рендера шаблонов (проверка подстановок)
2. Mock SMTP (интерфейс вокруг отправки) + тест consumer потока
3. Интеграционные тесты: локальный Kafka + MailHog контейнер
4. Тестирование отказов (сломанный шаблон / отсутствующий файл)

## Troubleshooting
| Симптом | Возможная причина | Решение |
|---------|-------------------|---------|
| panic на старте | Не задана обязательная переменная | Проверьте `.env` |
| Письма не доходят | Неверный SMTP пароль / блокировка провайдера | Сгенерируйте корректный пароль приложения, проверьте логи |
| Пустое тело письма | Некорректные данные `data` для шаблона | Сверьте ключи с шаблоном |
| Логотип не отображается | Отсутствует `icon.png` или нет `cid:logo` в HTML | Добавьте изображение и ссылку `cid:logo` |
| Задержки отправки | Проблемы Kafka или SMTP connectivity | Проверьте consumer lag и сетевые настройки |

## Возможные улучшения / Roadmap
- Поддержка дополнительных каналов (SMS, WebPush)
- Метрики + экспортер (Prometheus)
- Набор готовых шаблонов для других сценариев (изменение пароля, уведомления безопасности)
- Опциональный ретрай + backoff внутри сервиса
- Поддержка нескольких поставщиков (fallback SMTP/API)
- Поддержка i18n (многоязычные шаблоны)

## Лицензия
### [MIT](https://github.com/Anabol1ks/LinkVault-micro/blob/master/LICENSE)


