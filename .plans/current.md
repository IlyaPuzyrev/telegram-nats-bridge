# Текущий план: Telegram Client

## Цель
Создать клиент для Telegram Bot API с поддержкой getUpdates и long polling.

## Статус: ✅ ВЫПОЛНЕНО

## Что сделано

### 1. **Структуры данных** (telegram_types.go) ✅
- [x] Update - основная структура для получения обновлений
- [x] Message - сообщение от пользователя
- [x] Chat - информация о чате
- [x] User - информация о пользователе
- [x] MessageEntity - сущности в тексте
- [x] PhotoSize - размеры фото
- [x] GetUpdatesResponse - ответ API

### 2. **Интерфейс клиента** (telegram_client.go) ✅
- [x] TelegramClientInterface с методами:
  - [x] GetUpdates(ctx, offset) - с контекстом
  - [x] GetUpdatesWithTimeout(ctx, offset, timeout) - long polling
  - [x] GetBotInfo(ctx) - информация о боте
  - [x] GetMe(ctx) - алиас для GetBotInfo

### 3. **Реализация клиента** ✅
- [x] Используется resty/v2 для HTTP запросов
- [x] Base URL: https://api.telegram.org/bot<token>
- [x] Метод getUpdates с поддержкой offset и timeout
- [x] Обработка ошибок API (HTTP статусы и Telegram error codes)
- [x] Логирование через slog
- [x] Retry механизм (3 попытки)

### 4. **Интеграция с main** ✅
- [x] Чтение TELEGRAM_BOT_TOKEN из env
- [x] Создание клиента в команде run
- [x] Тестовый вызов GetMe для проверки подключения
- [x] Тестовый цикл GetUpdates с graceful shutdown
- [x] Обработка SIGINT/SIGTERM

## Созданные файлы
- ✅ telegram_types.go - структуры API
- ✅ telegram_client.go - клиент и интерфейс
- ✅ Обновление main.go - интеграция клиента
- ✅ Обновление go.mod/go.sum - зависимости

## Результат
Клиент готов к использованию. Можно получать реальные сообщения из Telegram для анализа структуры данных и проектирования маппинга в NATS subjects.

## Следующий шаг
Протестировать клиент с реальным ботом и получить примеры сообщений для проектирования конфигурации маппинга событий.
