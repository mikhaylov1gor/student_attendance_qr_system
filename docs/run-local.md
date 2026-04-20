# Локальный запуск

Пошаговая инструкция для чистого клона. Команды в блоке `bash` выполняются
в Git Bash (Windows) или обычном bash/zsh (Linux/macOS).

## Требования

| Компонент | Версия | Зачем |
|---|---|---|
| Go | 1.25+ | компилятор бэкенда |
| Docker Desktop / Docker Engine | любая с `docker compose` | Postgres 16 в контейнере |
| Git Bash (только Windows) | любая | `scripts/*.sh` рассчитаны на bash |
| `openssl` | любая | генерация ключей |

Make **не нужен** — все скрипты запускаются через `bash scripts/<name>.sh`.

## Шаги

### 1. Склонировать и зайти в каталог

```bash
git clone <url> attendance
cd attendance
```

### 2. Скопировать `.env.example` и сгенерировать секреты

```bash
cp .env.example .env
```

Открой `.env` и замени три плейсхолдера (`change_me_*`) на случайные значения:

```bash
# PII — 32 байта base64 для AES-256-GCM (шифрование ФИО)
openssl rand -base64 32

# JWT — как минимум 48 байт, подпись HS256
openssl rand -base64 48
```

Подставь выводы в `PII_ENCRYPTION_KEY` и `JWT_ACCESS_SECRET`.

`DATABASE_DSN` менять не нужно — он соответствует `docker-compose.yml`.

### 3. Поднять Postgres

```bash
docker compose up -d db
```

Ждём `healthy`:
```bash
docker ps --format '{{.Names}} {{.Status}}'
# attendance-db  Up 10 seconds (healthy)
```

### 4. Применить миграции и seed-данные

```bash
bash scripts/migrate-up.sh
```

Ожидаемый вывод: `OK 0001_init.sql`, `OK 0002_seed_dev.sql`, `OK 0003_refresh_tokens.sql`.

Seed создаёт 3 курса, 3 группы, 3 потока, 3 аудитории и 2 политики безопасности.
Пользователи **не сидятся** (требуют argon2id + AES-GCM, поэтому создаются через CLI).

### 5. Создать первого администратора

```bash
go run ./cmd/seed-admin \
    --email admin@tsu.ru \
    --password 'Pa55w0rd-strong!' \
    --last 'Михайлов' --first 'Игорь' --middle 'Александрович'
```

Вывод: `✓ user создан (role=admin)`.

### 6. Запустить API

```bash
go run ./cmd/api
```

Должно появиться:
```
level=INFO msg="rotator: bootstrap" resumed=0
level=INFO msg="http server listening" addr=:8080
```

### 7. Быстрая проверка

В другом терминале:

```bash
curl localhost:8080/healthz
# {"status":"ok","version":"dev"}

curl -s -X POST localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@tsu.ru","password":"Pa55w0rd-strong!"}' | head -c 200
# {"access_token":"eyJhbGci...","refresh_token":"...","expires_in":900,...}
```

Дальше — `docs/api-examples.md` с готовыми рецептами для основных флоу.

## Частые проблемы

| Симптом | Причина | Решение |
|---|---|---|
| `connection refused` на 5432 | Docker Desktop не запущен | Старт Docker, повтор `docker compose up -d db` |
| `DATABASE_DSN is required` | нет `.env` | `cp .env.example .env` |
| `key must be 32 bytes` | не поменял `change_me_base64_32bytes` | `openssl rand -base64 32` → заменить |
| `JWT_ACCESS_SECRET must be ≥32 bytes` | короткий секрет | `openssl rand -base64 48` |
| `invalid_filter` на reports | забыт `session_id`/`group_id`/`course_id` | добавить один из них в query |
| `audit verify` падает после прямой правки БД | это не баг, это фича | Именно для этого hash-chain и сделан — см. `docs/architecture/er-model.md` |

## Дополнительные сценарии

### Создать teacher/student

```bash
# teacher
go run ./cmd/seed-admin --role teacher \
    --email teacher@tsu.ru --password 'Teacher-pa55!' \
    --last 'Препод' --first 'Иван'

# student (нужен group-id из seed'а — см. миграцию 0002)
go run ./cmd/seed-admin --role student \
    --email student@tsu.ru --password 'Student-pa55!' \
    --last 'Студентов' --first 'Иван' \
    --group-id 22222222-2222-2222-2222-222222222201
```

### Откатить миграции

```bash
bash scripts/migrate-down.sh  # одну назад
# или
go run ./cmd/migrate reset     # всё до нуля
```

### Пересобрать БД с нуля

```bash
docker compose down -v           # удаляет volume
docker compose up -d db
bash scripts/migrate-up.sh
# + пересоздать admin через seed-admin
```

### Проверить состояние audit-цепочки

```bash
curl -s -X POST localhost:8080/api/v1/audit/verify \
    -H "Authorization: Bearer $ADMIN_JWT"
# {"ok":true,"total_entries":42}
```

Если `ok:false` + `first_broken_id` — значит, кто-то правил `audit_log` в
обход сервиса (ручной UPDATE в psql). Это **ожидаемое** поведение детектора.

## Структура проекта

```
cmd/
  api/              — HTTP API (основной бинарь)
  migrate/          — goose CLI обёртка
  seed-admin/       — создание первого пользователя
  wstest/           — dev-утилита для проверки WebSocket
internal/
  domain/           — чистые типы + порты (без gorm/http)
  application/      — use case'ы
  infrastructure/   — Gorm, crypto, HTTP, WS
  platform/         — нейтральная инфра (clock, logging)
migrations/         — SQL-миграции (goose)
docs/               — проектная документация
scripts/            — bash-скрипты (migrate, run, test, lint)
```

## Что дальше

- [docs/api-examples.md](api-examples.md) — curl-рецепты
- [docs/architecture/er-model.md](architecture/er-model.md) — модель данных
- [docs/db-migrations.md](db-migrations.md) — правила работы с миграциями
- [docs/implementation-plan.md](implementation-plan.md) — история разработки по этапам
