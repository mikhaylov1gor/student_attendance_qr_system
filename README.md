# Attendance — система учёта посещаемости с QR

Курсовая работа, ВШИТ ТГУ (09.03.04 Программная инженерия).

## Что это

Система отметки студентов через QR-коды с подключаемыми механизмами защиты
(QR-TTL по счётчику ротации, геолокация, Wi-Fi BSSID). Полная цепочка:
преподаватель запускает сессию → на экране крутится подписанный QR → студент
сканирует мобилкой → бэкенд прогоняет security-policy → отметка попадает в
tamper-evident журнал с hash-chain.

Архитектурные точки сложности:

1. **Pluggable Security Mechanisms** (Strategy pattern + runtime Policy Engine).
2. **QR-ротация** со stateless HMAC-токенами + WebSocket broadcast.
3. **Tamper-evident audit log** (SHA-256 hash chain + детерминированный JSON).
4. **Clean Architecture** в 4 слоя: domain → application → infrastructure.

Подробнее — [backend/docs/architecture/](backend/docs/architecture/) и
[backend/docs/implementation-plan.md](backend/docs/implementation-plan.md).

## Структура монорепы

```
kursovaya/
├── backend/              Go API (готов)
│   ├── cmd/              бинарники (api, migrate, seed-admin, wstest)
│   ├── internal/         domain + application + infrastructure
│   ├── migrations/       goose SQL
│   ├── docs/             архитектурные и OP-доки (run-local, api-examples)
│   └── scripts/          bash-шорткаты
├── frontend/
│   ├── teacher/          Vite + React + TS + Mantine (в разработке)
│   └── admin/            Vite + React + TS + Mantine (в разработке)
└── docs/
    └── frontend-plan.md  дизайн фронта (что/зачем/как)
```

Мобильный клиент (React Native + Expo, Android-only) выйдет в отдельную итерацию.

## Быстрый старт

```bash
# Бэкенд
cd backend
cp .env.example .env
# → заполнить PII_ENCRYPTION_KEY и JWT_ACCESS_SECRET через openssl rand
docker compose up -d db
bash scripts/migrate-up.sh
go run ./cmd/seed-admin --email admin@tsu.ru --password 'Pa55w0rd!' \
    --last Михайлов --first Игорь
go run ./cmd/api
# API слушает :8080

# Фронт (когда будет готов)
cd ../frontend/teacher
npm install
npm run dev
# SPA на :5173
```

Полная инструкция: [backend/docs/run-local.md](backend/docs/run-local.md).
Curl-рецепты по всем эндпоинтам: [backend/docs/api-examples.md](backend/docs/api-examples.md).

## Статус

- ✅ Бэкенд — 13 этапов плана, все чекпоинты зелёные.
- 🚧 Фронтенд — стартует следующим этапом ([docs/frontend-plan.md](docs/frontend-plan.md)).
- ⏸ Мобилка — после фронта.

## Лицензия

Код учебный. Использование — с указанием автора.
