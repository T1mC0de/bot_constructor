# Конструктор ботов

Продавец загружает анкету, Telegram-бот проводит по ней покупателя и сохраняет ответы как заявку.

## Backend

- `backend/core` — API для продавцов: загрузка анкеты, список заявок. Данные хранятся в JSON-файлах.
- `backend/bot` — Telegram-бот: задаёт вопросы анкеты и отправляет заявку в core.

## Запуск

```bash
cd backend
cp .env.example .env   # вписать TELEGRAM_BOT_TOKEN
make run
```

Загрузить анкету: `core/scripts/upload_survey.sh core/examples/survey_v1.json`.
Скачать заявки: `core/scripts/download_orders.sh`.

Тесты: `make test`.
