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
| `github.com/spf13/cobra` | CLI фреймворк |
| `github.com/spf13/viper` | Чтение YAML конфига |
| `github.com/stretchr/testify` | Тестирование |
| `github.com/expr-lang/expr` | Язык выражений для маршрутизации |

## Конфигурация

**Env переменные:**
- `TELEGRAM_BOT_TOKEN` — токен Telegram бота
- `NATS_URL` — URL NATS сервера

**YAML конфиг:** путь передаётся через флаг `--config`

```yaml
# Режим маршрутизации: "first" - первое совпадение, "all" - все совпадения
mode: "first"

# Правила маршрутизации
routes:
  - condition: "update.message != nil"  # условие на expr
    subject:
      type: "string"  # или "expr"
      value: "telegram.messages"  # тема или expr-программа

# Опционально: можно задать здесь вместо env
# telegram_token: "..."
# nats_url: "nats://localhost:4222"
```

**Приоритет:**
- `mode` и `routes` — только из YAML
- `telegram_token` и `nats_url` — из YAML или env (viper объединяет)

### Маршрутизация сообщений

Bridge использует [Expr](https://github.com/expr-lang/expr) для маршрутизации updates.

**Режимы:**
- `mode: "first"` — отправить на subject первого matched правила
- `mode: "all"` — отправить на subject каждого matched правила

**Структура правила:**
- `condition` — выражение на Expr, возвращающее bool
- `subject.type` — `"string"` (статическая тема) или `"expr"` (динамическая)
- `subject.value` — тема или expr-программа

**Примеры:**
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
