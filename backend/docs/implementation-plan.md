# План реализации бэкенда

Этот план охватывает только Go-бэкенд: от пустой директории до API, готового к интеграции с мобильным клиентом и двумя SPA. Каждый этап — атомарный чекпоинт, который можно проверить руками и закоммитить.

---

## Этап 1. Каркас проекта и инфраструктура разработки

**Чекпоинт:** `go run ./cmd/api` поднимает HTTP-сервер на `:8080`, `/healthz` отвечает `200 OK` с JSON `{"status":"ok","version":"dev"}`. `docker compose up -d db` поднимает PostgreSQL, `psql` коннектится.

**Что делаем:**
1. `go mod init github.com/mikhaylov-ii/attendance` + `.gitignore`, `.editorconfig`.
2. Структура директорий (см. раздел «Решения» ниже).
3. `cmd/api/main.go` — bootstrap: загрузка конфига, логгер, router, graceful shutdown через `signal.NotifyContext`.
4. `internal/config/config.go` — типизированный конфиг через `caarlos0/env`.
5. `internal/platform/logging/logging.go` — обёртка над `slog` (JSON в prod, текст в dev).
6. `internal/platform/httpserver/server.go` — `http.Server` с нормальными timeouts.
7. Handler `/healthz` в `internal/platform/httpserver/health.go`.
8. `docker-compose.yml` с сервисом `db` (Postgres 16, порт 5432, volume, `POSTGRES_DB=attendance`).
9. `scripts/` — `run.sh`, `migrate-up.sh`, `migrate-down.sh`, `migrate-new.sh`, `test.sh`, `lint.sh`. Запускаются из Git Bash. (Make на Windows не стандартен, `scripts/*.sh` универсальнее.)
10. `.env.example` с полным набором переменных.

**Зависимости:** нет.

---

## Этап 2. Миграции и модель данных в БД

**Чекпоинт:** `scripts/migrate-up.sh` применяет все миграции; `\dt` в psql показывает 12 таблиц; все enum-типы, partial unique index на `security_policies.is_default`, GIN-индекс на `classrooms.allowed_bssids` созданы; `migrate-down.sh` всё сносит. Dev-seed загружен.

**Что делаем:**
1. Подключить `goose` (как библиотека + отдельная утилита `cmd/migrate/main.go`, которую зовут скрипты).
2. `migrations/0001_init.sql` — все таблицы из ER. Одной миграцией.
3. ENUM-типы: `user_role`, `session_status`, `preliminary_status`, `final_status`, `check_status` как Postgres ENUM.
4. Partial unique index: `CREATE UNIQUE INDEX uniq_default_policy ON security_policies (is_default) WHERE is_default = true AND deleted_at IS NULL`.
5. Индексы: `sessions.teacher_id`, `sessions.starts_at`, unique `(session_id, student_id)` на `attendance_records`, `audit_log.occurred_at`, `audit_log.actor_id`.
6. `migrations/0002_seed_dev.sql` — dev seed.
7. `docs/db-migrations.md` — правила работы с миграциями.

**Зависимости:** этап 1.

**Риски:** AutoMigrate от Gorm отклонён — не умеет ENUM и partial index.

---

## Этап 3. Доменный слой: типы и интерфейсы портов

**Чекпоинт:** `go build ./internal/domain/...` зелёный. В `domain` — только чистые типы и интерфейсы, без gorm/net/http/database/sql. `go list -deps` подтверждает.

**Что делаем:**
1. `internal/domain/user` — `User`, `Role`, `FullName` (value object).
2. `internal/domain/group`, `stream`, `course`, `classroom` — entity-типы.
3. `internal/domain/session` — `Session`, `SessionStatus`, `QRSecret`.
4. `internal/domain/attendance` — `AttendanceRecord`, статусы, `SecurityCheckResult`, `CheckStatus`.
5. `internal/domain/policy` — `SecurityPolicy`, `MechanismsConfig`.
6. `internal/domain/audit` — `AuditEntry`.
7. Порты-репозитории: `UserRepository`, `SessionRepository`, `AttendanceRepository`, `PolicyRepository`, `AuditRepository`, etc.
8. Порт `SecurityCheck`:
   ```go
   type SecurityCheck interface {
       Name() string
       Check(ctx, input) (CheckStatus, details map[string]any, err error)
   }
   ```
9. Крипто-порты: `PasswordHasher`, `FieldEncryptor`, `TokenSigner`, `QRTokenCodec`.

**Зависимости:** этап 2.

**Риски:** gorm/json-теги в domain — запрещены. Теги только на infra-моделях.

---

## Этап 4. Infrastructure: Gorm-репозитории и криптопримитивы

**Чекпоинт:** вручную создать политику, пользователя, курс, аудиторию — всё сохраняется, читается обратно, ФИО plaintext в приложении, бинарь в БД. `password_hash` в формате `$argon2id$v=19$...`.

**Что делаем:**
1. `internal/infrastructure/db/gorm.go` — соединение, pool, логгер Gorm → slog.
2. `internal/infrastructure/db/models/` — Gorm-модели с тегами, колонки `full_name_ciphertext`, `full_name_nonce`. Mapper'ы domain↔model (ручные).
3. `internal/infrastructure/db/repo/` — реализации репозиториев, один файл на репо.
4. `internal/infrastructure/crypto/argon2id.go` — OWASP 2024 baseline (m=64MB, t=3, p=2).
5. `internal/infrastructure/crypto/aesgcm.go` — ключ 32B base64 из env, nonce 12B crypto/rand.
6. `internal/infrastructure/crypto/hmac.go` — HMAC-SHA256 (для QR).

**Зависимости:** этапы 2, 3.

**Риски:** Gorm hooks для шифрования отклонены — прячут криптооперацию. Явный mapper.

---

## Этап 5. Application layer: аутентификация

**Чекпоинт:** через curl login/refresh/logout/me работают end-to-end. Access 15 мин, refresh 7 дней с ротацией.

**Что делаем:**
1. Миграция `0003_refresh_tokens.sql` (id, user_id, token_hash, issued_at, expires_at, revoked_at).
2. `internal/domain/auth` — типы, ошибки.
3. `internal/application/auth/usecase.go` — `Login`, `Refresh`, `Logout`, `CurrentUser`.
4. `internal/infrastructure/crypto/jwt.go` — HS256, claims `sub`, `role`, `exp`, `iat`, `jti`.
5. `internal/infrastructure/http/handlers/auth.go`.
6. `internal/infrastructure/http/middleware/auth.go` — парсит Bearer, кладёт `user_id`, `role` в context.
7. `middleware/requirerole.go`.
8. DTO с validator-тегами.
9. Router: `/api/v1/auth/*` публичные, остальное за auth.
10. Refresh хранится как SHA-256 хэш; при ротации старый revoked.

**Зависимости:** этапы 3, 4.

---

## Этап 6. Security Mechanisms + Policy Engine (точка сложности #1)

**Чекпоинт:** админ CRUD политик через REST; `set-default` транзакционно. Первые unit-тесты Policy Engine.

**Что делаем:**
1. `internal/domain/policy/engine.go` — `Engine{checks []SecurityCheck}`, `Evaluate(ctx, config, input) []Result`.
2. `internal/domain/policy/preliminary.go` — `DerivePreliminaryStatus`: all passed/skipped → accepted; any failed → needs_review; invalid QR → rejected (только в attendance).
3. `internal/domain/policy/checks/`:
   - `qrttl.go` — counter/TTL.
   - `geo.go` — haversine vs classroom + radius.
   - `wifi.go` — BSSID matching, skipped если нет данных.
4. `internal/application/policy/service.go` — CRUD.
5. `internal/infrastructure/http/handlers/policy.go`.
6. Валидация JSONB конфига через validator.

**Зависимости:** этапы 3, 4, 5.

**Риски:** регистрация механизмов — явный слайс в composition root.

---

## Этап 7. Аудит hash-chain (точка сложности #3)

**Чекпоинт:** любой insert через сервис строит правильную цепочку; `POST /audit/verify` возвращает `{ok:true}`; после ручной порчи одной записи в psql — `{ok:false, first_broken_id:N}`.

**Что делаем:**
1. `internal/domain/audit/canonical.go` — детерминированный JSON (сортировка ключей, RFC3339Nano UTC, без пробелов).
2. `internal/domain/audit/hash.go` — `ComputeRecordHash(prev, payload, occurredAt)`.
3. `internal/application/audit/service.go` — `Append`:
   - Транзакция + `pg_advisory_xact_lock(hashtext('audit_log'))`.
   - SELECT последней, вычисление, INSERT.
4. `internal/application/audit/verify.go` — пересчёт цепочки батчами.
5. Handlers `GET /audit` + `POST /audit/verify`.
6. Интеграция `Append` в use cases: auth, policy, session (этап 8), attendance (этап 9), users (этап 10). Всегда в той же транзакции, что основное действие.

**Зависимости:** этапы 3, 4, 5, 6.

**Риски:** canonical JSON — критичное место, тесты обязательны.

---

## Этап 8. Справочники и сессии (без QR)

**Чекпоинт:** админ создаёт catalog-сущности; преподаватель создаёт draft-сессию, `/start` генерит `qr_secret`. Инвариант «group принадлежит курсу» проверяется → 409 `groups_not_in_course_streams`.

**Что делаем:**
1. `internal/application/catalog/` — CRUD groups/streams/courses/classrooms, M:N stream↔group.
2. `internal/application/session/service.go` — Create/Update(draft)/Start/Close/Delete(draft)/Get/List.
3. В Start: `qr_secret = crypto/rand(32)`, `qr_counter = 0`, `status = active`, audit-append.
4. Handlers `catalog.go`, `sessions.go`.
5. DTO + validator.
6. `GET /sessions/:id/attendance` — заготовка.

**Зависимости:** этапы 5, 7.

**Риски:** soft delete для users/policies; hard для catalog с 409 при ссылках.

---

## Этап 9. QR-ротация + WebSocket + POST /attendance (точка сложности #2)

**Чекпоинт:**
- WS к `/ws/sessions/:id/teacher` — получение первого QR сразу, затем каждые `ttl` секунд.
- Закрытие WS при `/sessions/:id/close`.
- `POST /attendance` с валидным QR → 200 + preliminary_status; WS шлёт event преподавателю.
- Просроченный counter → qr_ttl=failed, но 200.
- Битый HMAC → 400 `invalid_qr_token`.
- Повторная отметка → 409 `already_submitted`.

**Что делаем:**
1. `internal/domain/session/qr.go` — `EncodeToken`/`DecodeAndVerify`. Формат: `base64url(session_id || counter || issued_at || hmac32)`.
2. `internal/application/qr/rotator.go` — goroutine per session, ticker, инкрементирует counter, публикует в hub.
3. `internal/application/hub/hub.go` — `session_id → []*TeacherConn`, Register/Unregister/Broadcast.
4. `internal/infrastructure/ws/` — `coder/websocket`, handler `/ws/sessions/:id/teacher`, JWT через subprotocol.
5. `internal/application/attendance/service.go` — Submit: decode → load session → HMAC → uniqueness → engine → transaction (records + audit) → broadcast.
6. Graceful shutdown: закрыть WS (1001), остановить rotator'ы.
7. Bootstrap: при старте поднять rotator'ы для `sessions.status=active`.

**Зависимости:** этапы 6, 7, 8.

**Риски:** rotator in-process → MVP только один инстанс API. `already_submitted` — unique index на БД.

---

## Этап 10. Admin CRUD + статистика студента

**Чекпоинт:** админ создаёт студентов с temp-паролем; reset-password работает; PATCH меняет ФИО (новый nonce), роль, group. Студент видит свои отметки и статистику.

**Что делаем:**
1. `internal/application/user/service.go` — CRUD + `ResetPassword` + `ListWithSearch`.
2. Генератор temp-пароля: 12 символов, crypto/rand, без неоднозначных.
3. Инвариант role↔current_group_id.
4. `internal/application/attendance/student_stats.go` — `ListMyAttendance`, `GetMyStats`.
5. Handlers `users.go`, `students_me.go`.
6. Audit-append для всех операций.

**Зависимости:** этапы 5, 7.

**Риски:** поиск по ФИО — in-memory после расшифровки.

---

## Этап 11. Отчёты Excel/CSV (синхронные)

**Чекпоинт:** `GET /reports/attendance.xlsx?...` отдаёт валидный xlsx; 1000 строк ≤ 2 сек.

**Что делаем:**
1. `internal/application/report/service.go` — `Attendance(ctx, filter, format)`.
2. `internal/infrastructure/report/xlsx.go` (excelize), `csv.go` (encoding/csv).
3. `handlers/reports.go` — стриминг в response.
4. Расшифровка ФИО перед записью.
5. Валидация: обязателен один из group/stream/course.

**Зависимости:** этапы 8, 9.

---

## Этап 12. Тесты

**Чекпоинт:** `scripts/test.sh` зелёный.

**Обязательные unit:**
1. `policy/engine_test.go` — passed/failed/skipped/disabled/error.
2. `audit/canonical_test.go` — детерминизм, сортировка, вложенность, time, nil.
3. `audit/hash_test.go` — test vectors.
4. `audit/verify_test.go` — валидная цепочка, детекция модификации middle-записи.
5. `session/qr_test.go` — encode/decode, подделка hmac, неверный session_id.
6. `policy/checks/geo_test.go` — haversine на известных координатах.

**Smoke (httptest):**
- `/auth/login` happy + wrong password.
- `/attendance` happy + bad hmac + already_submitted.
- `/audit` + `/audit/verify`.

**Не покрываем:** gorm-репо, admin CRUD, WebSocket (ручная проверка через `wscat`).

---

## Этап 13. Hardening и документация

**Чекпоинт:** API готов к фронтам.

1. Rate limiting на `/auth/login` (10/мин на IP).
2. CORS middleware с whitelist.
3. Request ID middleware + `X-Request-ID`.
4. Panic recover.
5. Timeouts на http.Server.
6. Централизованный error → HTTP mapper (sentinel-ошибки домена → коды).
7. `docs/run-local.md`, `docs/api-examples.md`.
8. Структурированные поля в логах.

---

# Решения, принятые перед стартом

| Вопрос | Решение | Обоснование |
|---|---|---|
| HTTP-роутер | **chi** | Идиоматичен для stdlib, не протекает в handlers, не ломает Clean Arch |
| Миграции | **goose** (библиотека + CLI-утилита) | SQL-миграции, AutoMigrate отклонён |
| Конфиг | **caarlos0/env** + `.env` через `godotenv` в dev | 12-factor, минимально |
| Логи | **log/slog** (stdlib Go 1.21+) | Без внешней зависимости |
| Валидация | **go-playground/validator/v10** | Стандарт |
| ORM | **Gorm** | Зафиксировано в теме курсовой |
| JWT | **HS256** | Монолит, единый секрет |
| WebSocket | **coder/websocket** | Современный API |
| Runner | **scripts/*.sh** (не Makefile) | Make на Windows не стандартен, скрипты универсальнее |
| Docker | `db` в compose, приложение на хосте | Скорость разработки |
| AES-ФИО | Две колонки `ciphertext + nonce` | Прозрачнее в БД, соответствует ER |
| Audit-transaction | `pg_advisory_xact_lock(hashtext('audit_log'))` | Простой, надёжный |
| Rotator | In-memory + bootstrap при старте | Один инстанс API в MVP |

## Структура директорий

```
.
├── cmd/
│   ├── api/
│   │   └── main.go                    # composition root
│   └── migrate/
│       └── main.go                    # goose CLI-обёртка
├── internal/
│   ├── domain/                        # Layer 1: чистые типы + порты
│   │   ├── user/
│   │   ├── session/
│   │   ├── attendance/
│   │   ├── policy/
│   │   │   ├── engine.go
│   │   │   ├── preliminary.go
│   │   │   └── checks/
│   │   │       ├── qrttl.go
│   │   │       ├── geo.go
│   │   │       └── wifi.go
│   │   ├── audit/
│   │   ├── auth/
│   │   ├── catalog/                   # groups, streams, courses, classrooms
│   │   └── ports.go
│   ├── application/                   # Layer 2: use cases
│   │   ├── auth/
│   │   ├── user/
│   │   ├── session/
│   │   ├── attendance/
│   │   ├── policy/
│   │   ├── audit/
│   │   ├── report/
│   │   ├── catalog/
│   │   ├── hub/                       # in-process pub/sub для WS
│   │   └── qr/                        # rotator
│   ├── infrastructure/                # Layer 3: адаптеры
│   │   ├── config/
│   │   ├── db/
│   │   │   ├── gorm.go
│   │   │   ├── models/
│   │   │   └── repo/
│   │   ├── crypto/
│   │   │   ├── argon2id.go
│   │   │   ├── aesgcm.go
│   │   │   ├── hmac.go
│   │   │   └── jwt.go
│   │   ├── http/
│   │   │   ├── router.go
│   │   │   ├── middleware/
│   │   │   ├── handlers/
│   │   │   └── dto/
│   │   └── ws/
│   └── platform/                      # нейтральная инфра
│       ├── logging/
│       └── clock/
├── migrations/
├── scripts/
├── docs/
├── docker-compose.yml
├── go.mod
└── .env.example
```
