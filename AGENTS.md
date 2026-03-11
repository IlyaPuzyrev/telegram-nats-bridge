# AGENTS.md

## Команды разработки

Проект использует Taskfile для автоматизации:

- `task build` — сборка бинарника
- `task test` — запуск тестов (`go test ./...`)
- `task run` — запуск bridge с config.yaml
- `task check-bot` — проверка бота и вывод updates в JSON

Запуск утилиты:
```bash
./telegram-nats-bridge run --config config.yaml
./telegram-nats-bridge check bot --config config.yaml
```

## Локальное тестирование с NATS

Для локального тестирования используется Docker Compose с NATS сервером:

```bash
# Запуск NATS сервера
task nats-up

# Проверка статуса
open http://localhost:8222

# Подписка на тему (в отдельном терминале)
task nats-box -- nats sub telegram.updates
# или с другой темой:
docker run --rm -it --network host natsio/nats-box nats sub "your.subject"

# Запуск bridge (в другом терминале)
task run

# Остановка NATS
task nats-down
```

**Docker Compose:** Используется `docker-compose.yaml`.
- Порт 4222: клиентские подключения
- Порт 8222: HTTP мониторинг

## Структура проекта

Плоская структура, один общий package. Все файлы в корне проекта.

## Зависимости

| Библиотека | Назначение |
|------------|------------|
| `github.com/go-resty/resty/v2` | HTTP клиент для Telegram API |
| `github.com/nats-io/nats.go` | NATS клиент |
| `github.com/segmentio/kafka-go` | Kafka клиент |
| `github.com/spf13/cobra` | CLI фреймворк |
| `github.com/spf13/viper` | Чтение YAML конфига |
| `github.com/stretchr/testify` | Тестирование |
| `github.com/expr-lang/expr` | Язык выражений для маршрутизации |
| `golang.org/x/sync/errgroup` | Конкурентная компиляция expr программ |

## Конфигурация

**Env переменные:**
- `TELEGRAM_BOT_TOKEN` — токен Telegram бота
- `NATS_URL` — URL NATS сервера (когда broker: "nats")
- `KAFKA_BROKERS` — адреса Kafka брокеров (когда broker: "kafka"), формат: "host1:port1,host2:port2"

**YAML конфиг:** путь передаётся через флаг `--config`

```yaml
# Выбор брокера: "nats" (по умолчанию) или "kafka"
broker: "nats"

# Настройки NATS (обязательно если broker: "nats")
nats:
  url: "nats://localhost:4222"
  engine: "core"  # "core" или "jetstream"
  # jetstream:  # (если engine: "jetstream")
  #   stream_config: "./stream-config.json"

# Настройки Kafka (обязательно если broker: "kafka")
kafka:
  brokers:
    - "localhost:9092"
  # async: false           # асинхронная отправка
  # ack_required: -1       # -1=all, 0=none, 1=leader
  # batch_size: 0
  # batch_bytes: 1048576

# Режим маршрутизации: "first" - первое совпадение, "all" - все совпадения
mode: "first"

# Количество воркеров для конкурентной обработки routes (по умолчанию: 5)
route_workers: 5

# Количество воркеров для конкурентной публикации в брокер (по умолчанию: 5)
publish_workers: 5

# Таймаут (сек) для graceful shutdown publisher (по умолчанию: 10)
publish_shutdown_timeout: 10

# Правила маршрутизации
routes:
  # Для NATS:
  - condition: "update.message != nil"  # условие на expr
    subject:
      type: "string"  # или "expr"
      value: "telegram.messages"  # тема или expr-программа

  # Для Kafka:
  # - condition: "update.message != nil"
  #   topic:
  #     type: "string"  # или "expr"
  #     value: "telegram.messages"
  #   key:  # опционально
  #     type: "expr"
  #     value: "sprintf(\"%v\", update.message.from.id)"

# Опционально: можно задать здесь вместо env
# telegram_token: "..."
```

**Приоритет:**
- `broker`, `mode`, `routes`, `route_workers`, `publish_workers`, `publish_shutdown_timeout`, `nats`, `kafka` — только из YAML
- `telegram_token` — из YAML или env
- `nats.url` — из YAML или NATS_URL env
- `kafka.brokers` — из YAML или KAFKA_BROKERS env

### JetStream

При использовании `engine: "jetstream"` bridge публикует сообщения в JetStream стрим вместо Core NATS.

**Преимущества JetStream:**
- Персистентность сообщений на диске
- At-least-once доставка
- Возможность перезапуска consumer и получения пропущенных сообщений
- Рерайт исторических сообщений

**Конфигурация stream:**
- Путь к JSON файлу задаётся в `jetstream.stream_config`
- Формат: [jetstream.StreamConfig](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#StreamConfig)
- При старте bridge вызывает `CreateOrUpdateStream` — создаёт или обновляет стрим

**Пример stream-config.json:**
```json
{
  "Name": "TELEGRAM",
  "Description": "Stream for Telegram bot updates",
  "Subjects": ["telegram.>"],
  "Retention": "limits",
  "Storage": "file",
  "Replicas": 1
}
```

**Docker Compose для JetStream:**
```yaml
services:
  nats:
    image: nats:latest
    command: ["-js"]  # включить JetStream
```

### Kafka

При использовании `broker: "kafka"` bridge публикует сообщения в Kafka топики.

**Настройки Kafka:**
- `kafka.brokers` — список адресов Kafka брокеров
- `kafka.async` — асинхронная отправка (по умолчанию: false)
- `kafka.ack_required` — уровень подтверждения: -1=all, 0=none, 1=leader (по умолчанию: -1)
- `kafka.batch_size` — количество сообщений в батче (по умолчанию: 0)
- `kafka.batch_bytes` — максимальный размер батча в байтах (по умолчанию: 1048576)

**Docker Compose для Kafka:**
```yaml
services:
  kafka:
    image: confluentinc/cp-kafka:latest
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
```

## Локальное тестирование с Kafka

Для локального тестирования используется Docker Compose с Kafka:

```bash
# Запуск Kafka
task kafka-up

# Подписка на топик (в отдельном терминале)
docker run --rm -it --network host confluentinc/cp-kafka:latest kafka-console-consumer --topic telegram.messages --from-beginning --bootstrap-server localhost:9092

# Запуск bridge с Kafka конфигом (в другом терминале)
task run

# Остановка Kafka
task kafka-down
```

### Маршрутизация сообщений

Bridge использует [Expr](https://github.com/expr-lang/expr) для маршрутизации updates.

**Режимы:**
- `mode: "first"` — отправить на subject/topic первого matched правила
- `mode: "all"` — отправить на subject/topic каждого matched правила

**Структура правила:**
- `condition` — выражение на Expr, возвращающее bool
- `subject` — (для NATS) тема:
  - `subject.type` — `"string"` (статическая) или `"expr"` (динамическая)
  - `subject.value` — тема или expr-программа
- `topic` — (для Kafka) топик:
  - `topic.type` — `"string"` (статический) или `"expr"` (динамический)
  - `topic.value` — топик или expr-программа
- `key` — (для Kafka, опционально) ключ:
  - `key.type` — `"string"` или `"expr"`
  - `key.value` — ключ или expr-программа

**Примеры для NATS:**
```yaml
# Сообщения от конкретного пользователя по ID
- condition: "update.message?.from?.id != nil"
  subject:
    type: "expr"
    value: "sprintf(\"telegram.messages.%v\", update.message.from.id)"

# Отредактированные сообщения
- condition: "update.edited_message != nil"
  subject:
    type: "string"
    value: "telegram.edited"

# Callback запросы
- condition: "update.callback_query != nil"
  subject:
    type: "string"
    value: "telegram.callbacks"

# Все сообщения (общий канал)
- condition: "update.message != nil"
  subject:
    type: "string"
    value: "telegram.messages"
```

**Примеры для Kafka:**
```yaml
# Сообщения в конкретный топик
- condition: "update.message != nil"
  topic:
    type: "string"
    value: "telegram.messages"

# Сообщения с динамическим топиком по ID пользователя
- condition: "update.message != nil"
  topic:
    type: "expr"
    value: "sprintf(\"telegram.messages.%v\", update.message.from.id)"

# Сообщения с ключом для партиционирования
- condition: "update.message != nil"
  topic:
    type: "string"
    value: "telegram.messages"
  key:
    type: "expr"
    value: "sprintf(\"%v\", update.message.from.id)"

# Callback запросы
- condition: "update.callback_query != nil"
  topic:
    type: "string"
    value: "telegram.callbacks"
```

**Доступные операторы в expr:**
- Safe navigation: `?.` (например, `update.message?.from?.id` — вернёт nil если любая часть = nil)
- Сравнение: `==`, `!=`, `>`, `<`, `>=`, `<=`
- Логика: `and`, `or`, `not`
- Доступ к полям: точечная нотация (`update.message.from.id`)

**Доступные функции в expr:** `sprintf`

**Поведение:** Update, не подходящий ни под одно правило, игнорируется.

## CLI

Команды:
- `run` — запуск bridge (требует `--config`)
- `check bot` — проверка бота и вывод updates (требует `--config`)

Graceful shutdown реализован через механизмы cobra.

## Логирование

Используется `log/slog` из стандартной библиотеки Go.

## Тестирование

Запуск через `task test`, используем `testify` для assertions.

## Типы данных

**Update** представлен как `map[string]any` (раньше был struct), что позволяет гибко работать с любым содержимым от Telegram API без жёсткой типизации всех полей.

## Получение обновлений (Updates)

Bridge использует метод `getUpdates` Telegram Bot API для получения событий.

**Документация:** https://core.telegram.org/bots/api#getupdates

### Метод getUpdates

```
GET https://api.telegram.org/bot<token>/getUpdates
```

**Параметры:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `offset` | Integer | ID первого update для возврата (must be > max received update_id + 1) |
| `limit` | Integer | Количество updates (1-100), по умолчанию 100 |
| `timeout` | Integer | Timeout в секундах для long polling. По умолчанию 0 (short polling) |
| `allowed_updates` | Array of String | Список типов updates для получения |

**Long Polling:**
- Устанавливаем `timeout` > 0 (например, 30 секунд)
- Сервер держит соединение открытым, пока не появятся updates или не истечёт timeout
- Это позволяет получать updates почти realtime без webhook

**Update объект:**

| Поле | Тип | Описание |
|------|-----|----------|
| `update_id` | Integer | Уникальный идентификатор update. Начинается с определенного положительного числа и увеличивается последовательно |
| `message` | Message | *Optional*. Новое входящее сообщение любого типа — текст, фото, стикер и т.д. |
| `edited_message` | Message | *Optional*. Новая версия сообщения, которое было отредактировано |
| `channel_post` | Message | *Optional*. Новый входящий пост в канале любого типа |
| `edited_channel_post` | Message | *Optional*. Новая версия поста в канале, который был отредактирован |
| `business_connection` | BusinessConnection | *Optional*. Бот был подключен или отключен от бизнес-аккаунта |
| `business_message` | Message | *Optional*. Новое сообщение от подключенного бизнес-аккаунта |
| `edited_business_message` | Message | *Optional*. Отредактированное сообщение от бизнес-аккаунта |
| `deleted_business_messages` | BusinessMessagesDeleted | *Optional*. Сообщения были удалены из бизнес-аккаунта |
| `message_reaction` | MessageReactionUpdated | *Optional*. Изменена реакция на сообщение. Бот должен быть администратором и явно указать `"message_reaction"` в allowed_updates |
| `message_reaction_count` | MessageReactionCountUpdated | *Optional*. Изменилось количество реакций на сообщение с анонимными реакциями |
| `inline_query` | InlineQuery | *Optional*. Новый входящий inline запрос |
| `chosen_inline_result` | ChosenInlineResult | *Optional*. Результат inline запроса, выбранный пользователем |
| `callback_query` | CallbackQuery | *Optional*. Новый входящий callback запрос |
| `shipping_query` | ShippingQuery | *Optional*. Новый входящий запрос доставки |
| `pre_checkout_query` | PreCheckoutQuery | *Optional*. Новый входящий предварительный запрос оплаты |
| `purchased_paid_media` | PaidMediaPurchased | *Optional*. Пользователь купил платную медиа с непустым payload |
| `poll` | Poll | *Optional*. Новое состояние опроса. Бот получает только updates об остановленных вручную опросах и опросах, отправленных ботом |
| `poll_answer` | PollAnswer | *Optional*. Пользователь изменил свой ответ в неанонимном опросе |
| `my_chat_member` | ChatMemberUpdated | *Optional*. Обновлен статус участника бота в чате |
| `chat_member` | ChatMemberUpdated | *Optional*. Обновлен статус участника в чате. Бот должен быть администратором и явно указать `"chat_member"` в allowed_updates |
| `chat_join_request` | ChatJoinRequest | *Optional*. Отправлен запрос на вступление в чат |
| `chat_boost` | ChatBoostUpdated | *Optional*. Добавлен или изменен буст чата |
| `removed_chat_boost` | ChatBoostRemoved | *Optional*. Буст был удален из чата |

**Примечания:**
- В каждом update может присутствовать **максимум одно** из optional полей
- Поля `chat_member`, `message_reaction`, `message_reaction_count` требуют явного указания в `allowed_updates`
- Если не указать `allowed_updates`, по умолчанию будут получены все типы кроме `chat_member`, `message_reaction` и `message_reaction_count`

**Важно:**
- После получения updates нужно обновлять `offset` (set offset = max(update_id) + 1)
- Updates хранятся на сервере до 24 часов
- Используем `allowed_updates` для фильтрации ненужных типов событий

### Webhook

Альтернативный способ получения updates — через webhook. Telegram отправляет POST-запросы на указанный URL при появлении новых событий.

**Установка webhook:**

```
POST https://api.telegram.org/bot<token>/setWebhook
```

**Параметры:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `url` | String | HTTPS URL для отправки updates. Пустая строка — удалить webhook |
| `max_connections` | Integer | Максимум одновременных соединений (1-100), по умолчанию 40 |
| `allowed_updates` | Array of String | Список типов updates для получения |
| `secret_token` | String | Секретный токен для верификации запросов (1-256 символов) |
| `drop_pending_updates` | Boolean | Удалить накопившиеся updates |

**Удаление webhook:**

```
POST https://api.telegram.org/bot<token>/deleteWebhook
```

**Параметры:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `drop_pending_updates` | Boolean | Удалить накопившиеся updates |

**Важно:**
- Webhook и getUpdates взаимоисключающие — нельзя использовать одновременно
- URL должен быть HTTPS
- Telegram отправляет POST с JSON телом, содержащим объект Update
- При неудаче (HTTP статус != 2xx) Telegram повторит запрос несколько раз
- Поддерживаемые порты: 443, 80, 88, 8443
