# Миграции БД

Инструмент — [`goose`](https://github.com/pressly/goose) как библиотека, завёрнутая в собственный CLI `cmd/migrate`. SQL-миграции лежат в `migrations/` и встраиваются в бинарь через `embed.FS` (`migrations/embed.go`), поэтому на prod их не нужно катать отдельным файлом — достаточно самого бинаря.

## Команды

Все команды запускаются из корня проекта. Shell-обёртки в `scripts/` просто проксируют вызовы на `go run ./cmd/migrate`.

| Команда | Что делает |
|---|---|
| `scripts/migrate-up.sh` | применяет все новые миграции |
| `scripts/migrate-down.sh` | откатывает **одну** последнюю миграцию |
| `scripts/migrate-status.sh` | список миграций + применена/не применена |
| `scripts/migrate-new.sh <name>` | создаёт новый пустой SQL-файл с следующим порядковым номером |

Также напрямую через `go run ./cmd/migrate`:

```bash
go run ./cmd/migrate up             # прогнать все
go run ./cmd/migrate up-by-one      # одну
go run ./cmd/migrate down           # откатить последнюю
go run ./cmd/migrate reset          # снести все
go run ./cmd/migrate status
go run ./cmd/migrate version
go run ./cmd/migrate create <name>  # создать новый файл
```

DSN берётся из `DATABASE_DSN` в `.env` (или из окружения, если env не задан).

## Формат миграции

Goose читает SQL-файлы с маркерами `-- +goose Up` и `-- +goose Down`:

```sql
-- +goose Up

CREATE TABLE foo (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid()
);

-- +goose Down

DROP TABLE IF EXISTS foo;
```

Для многострочных statements (тела функций, анонимные `DO $$ … $$`) используй блок `+goose StatementBegin/End`:

```sql
-- +goose StatementBegin
CREATE FUNCTION trigger_fn() RETURNS trigger AS $$
BEGIN
    ...
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
```

## Нумерация

Четырёхзначный префикс (`0001_`, `0002_`, …). `migrate-new.sh` находит максимальный существующий номер и прибавляет единицу.

## Правила

1. **Миграции неизменяемы после мержа**. Уже применённый `0001_init.sql` редактировать нельзя — делай новую миграцию.
2. **Обязательный `Down`**. Любая up-миграция должна иметь down, чтобы `reset` работал для локальной разработки. Для prod откат не предполагается — но в dev он частый.
3. **Одна миграция — одна атомарная смысловая единица**. Не лепить 10 несвязанных изменений в один файл.
4. **ENUM'ы добавляются как Postgres `CREATE TYPE ... AS ENUM`**. `ALTER TYPE ... ADD VALUE` в миграции — только в блоке `StatementBegin/End` и только после прочтения https://www.postgresql.org/docs/16/sql-altertype.html (нельзя внутри транзакции, если значение используется сразу).
5. **Partial / GIN-индексы — в миграциях, не через AutoMigrate.** Gorm AutoMigrate не умеет ни то, ни другое. Поэтому AutoMigrate не используется.
6. **Seed dev-данных — отдельная миграция** (`0002_seed_dev.sql`). В prod её не катают (в будущем — `-- +goose NO TRANSACTION` + флаг `GOOSE_SKIP_SEED`; пока deployment'а нет — не актуально).

## Текущий список миграций

- `0001_init.sql` — полная начальная схема: 12 таблиц, 4 ENUM, partial unique index на `security_policies.is_default`, GIN на `classrooms.allowed_bssids`, unique `(session_id, student_id)` на `attendance_records`, CHECK'и инвариантов.
- `0002_seed_dev.sql` — справочные данные: 3 курса, 3 группы, 3 потока, 3 аудитории, 2 политики безопасности (`default`, `qr_only`). Пользователи **не сидятся** — требуют argon2id + AES-GCM, создаются через API начиная с этапа 10.
