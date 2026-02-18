# AGENTS.md

## Команды разработки

Проект использует Taskfile для автоматизации:

- `task build` — сборка бинарника
- `task test` — запуск тестов (`go test ./...`)
- `task run` — запуск для разработки

Запуск утилиты:
```bash
./telegram-nats-bridge run --config config.yaml
```

## Структура проекта

Плоская структура, один общий package. Все файлы в корне проекта.

## Зависимости

| Библиотека | Назначение |
|------------|------------|
| `github.com/go-resty/resty/v2` | HTTP клиент для Telegram API |
| `github.com/nats-io/nats.go` | NATS клиент |
| `github.com/spf13/cobra` | CLI фреймворк |
| `github.com/spf13/viper` | Чтение env переменных |
| `github.com/stretchr/testify` | Тестирование |

## Конфигурация

**Env переменные:**
- `TELEGRAM_BOT_TOKEN` — токен Telegram бота
- `NATS_URL` — URL NATS сервера
- `NATS_CREDENTIALS` — опционально, путь к credentials файлу

**YAML конфиг:** путь передаётся через флаг `--config`, содержит маппинг событий в NATS subjects (структура будет определена позже).

## CLI

Одна команда: `run`. Graceful shutdown реализован через механизмы cobra.

## Логирование

Используется `log/slog` из стандартной библиотеки Go.

## Тестирование

Запуск через `task test`, используем `testify` для assertions.

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
