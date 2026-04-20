# API examples — curl-рецепты

Справочник по всем основным эндпоинтам. Для краткости предполагается:

```bash
API=http://localhost:8080
# ADMIN, TEACHER, STUDENT — access-токены после логина под разными ролями.
```

## Формат ошибок

Единый для всего API:

```json
{"error": {"code": "invalid_qr_token", "message": "qr token invalid or malformed"}}
```

- `code` — машинно-читаемый, стабильный (можно на клиенте `switch`'ить).
- `message` — человеку, может содержать детали (например, конкретные
  поля, провалившие валидацию).

Полный каталог кодов — в [httperr/registry.go](../internal/infrastructure/http/httperr/registry.go).

## Healthcheck

```bash
curl $API/healthz
# {"status":"ok","version":"dev"}
```

## Аутентификация

### Login

```bash
curl -s -X POST $API/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@tsu.ru","password":"Pa55w0rd-strong!"}'
```

Ответ:
```json
{
  "access_token":  "eyJ...",
  "refresh_token": "kaK...",
  "expires_in":    900,
  "expires_at":    "2026-04-20T20:15:00Z",
  "token_type":    "Bearer"
}
```

Rate-limit: 10 запросов в минуту с одного IP → `429 rate_limited` + `Retry-After`.

### Refresh (ротация)

```bash
curl -s -X POST $API/api/v1/auth/refresh \
    -H "Content-Type: application/json" \
    -d '{"refresh_token":"kaK..."}'
```

Старый refresh **сразу отзывается** — повтор даст `401 token_revoked`.

### Me

```bash
curl -s $API/api/v1/auth/me -H "Authorization: Bearer $ACCESS"
```

### Logout

```bash
curl -X POST $API/api/v1/auth/logout \
    -H "Authorization: Bearer $ACCESS" \
    -H "Content-Type: application/json" \
    -d '{"refresh_token":"kaK..."}'
# 204 No Content
```

## Каталог

Чтение — любой аутентифицированный. Мутации — только admin.

```bash
# Курсы
curl $API/api/v1/courses                             -H "Authorization: Bearer $ANY"
curl $API/api/v1/courses/<uuid>                      -H "Authorization: Bearer $ANY"
curl -X POST $API/api/v1/courses  -H "Authorization: Bearer $ADMIN" \
    -d '{"name":"Новый курс","code":"NEW-01"}'
curl -X PATCH $API/api/v1/courses/<uuid> -H "Authorization: Bearer $ADMIN" \
    -d '{"name":"Переименован"}'
curl -X DELETE $API/api/v1/courses/<uuid> -H "Authorization: Bearer $ADMIN"

# Группы
curl $API/api/v1/groups -H "Authorization: Bearer $ANY"
curl -X POST $API/api/v1/groups -H "Authorization: Bearer $ADMIN" \
    -d '{"name":"БПИ-241"}'

# Потоки (обязателен course_id)
curl "$API/api/v1/streams?course_id=<uuid>" -H "Authorization: Bearer $ANY"
curl -X POST $API/api/v1/streams -H "Authorization: Bearer $ADMIN" \
    -d '{"course_id":"<uuid>","name":"Поток А","group_ids":["<uuid>","<uuid>"]}'

# Аудитории
curl $API/api/v1/classrooms -H "Authorization: Bearer $ANY"
curl -X POST $API/api/v1/classrooms -H "Authorization: Bearer $ADMIN" \
    -d '{"building":"Главный","room_number":"201","latitude":56.47,"longitude":84.95,"radius_m":25,"allowed_bssids":["aa:bb:cc:dd:ee:01"]}'
```

## Пользователи (admin-only)

```bash
# List + search by ФИО (in-memory, после расшифровки)
curl "$API/api/v1/users?role=student&q=Иванов&limit=20" -H "Authorization: Bearer $ADMIN"

# Create — password опционален, если пуст → temp генерируется
curl -X POST $API/api/v1/users -H "Authorization: Bearer $ADMIN" \
    -d '{"email":"s@x.ru","role":"student","last":"Иванов","first":"Пётр","current_group_id":"<uuid>"}'
# { "user": {...}, "temp_password": "JcWUYbAeTzPE" }

# PATCH — частичный, clear_group снимает привязку
curl -X PATCH $API/api/v1/users/<uuid> -H "Authorization: Bearer $ADMIN" \
    -d '{"role":"teacher","clear_group":true}'

# Reset password
curl -X POST $API/api/v1/users/<uuid>/reset-password -H "Authorization: Bearer $ADMIN"
# {"temp_password":"NewTempXY12"}

# Delete (soft)
curl -X DELETE $API/api/v1/users/<uuid> -H "Authorization: Bearer $ADMIN"
```

## Политики безопасности (admin-only)

```bash
# List
curl $API/api/v1/policies -H "Authorization: Bearer $ADMIN"

# Create
curl -X POST $API/api/v1/policies -H "Authorization: Bearer $ADMIN" \
    -d '{
      "name":"strict",
      "mechanisms":{
        "qr_ttl":{"enabled":true,"ttl_seconds":5},
        "geo":{"enabled":true},
        "wifi":{"enabled":true,"required_bssids_from_classroom":true}
      }
    }'

# Set default (атомарно снимает флаг с предыдущей)
curl -X POST $API/api/v1/policies/<uuid>/set-default -H "Authorization: Bearer $ADMIN"

# Валидация
curl -X POST $API/api/v1/policies -H "Authorization: Bearer $ADMIN" \
    -d '{"name":"bad","mechanisms":{"qr_ttl":{"enabled":true,"ttl_seconds":1}}}'
# 400 invalid_config: "qr_ttl.ttl_seconds must be in [3, 120], got 1"
```

## Сессии (teacher + admin)

```bash
# Создать draft
curl -X POST $API/api/v1/sessions -H "Authorization: Bearer $TEACHER" \
    -d '{
      "course_id":"<uuid>",
      "classroom_id":"<uuid>",
      "starts_at":"2026-04-20T10:00:00Z",
      "ends_at":"2026-04-20T11:30:00Z",
      "group_ids":["<uuid>","<uuid>"]
    }'
# 201, status=draft, qr_ttl_seconds=10 (наследовано из default policy)

# Start: draft → active, перегенерация qr_secret
curl -X POST $API/api/v1/sessions/<sid>/start -H "Authorization: Bearer $TEACHER"

# Close: active → closed (rotator останавливается, WS-клиенты отваливаются)
curl -X POST $API/api/v1/sessions/<sid>/close -H "Authorization: Bearer $TEACHER"

# List с фильтрами
curl "$API/api/v1/sessions?course_id=<uuid>&status=active" -H "Authorization: Bearer $TEACHER"
```

## Отметка посещаемости (student-only)

```bash
curl -X POST $API/api/v1/attendance -H "Authorization: Bearer $STUDENT" \
    -d '{
      "qr_token":"<token, полученный из WS>",
      "geo_lat":56.469849,
      "geo_lng":84.948042,
      "bssid":"aa:bb:cc:dd:ee:01"
    }'
```

Успех:
```json
{
  "id":"<uuid>",
  "session_id":"<uuid>",
  "submitted_at":"2026-04-20T10:05:12Z",
  "preliminary_status":"accepted",  // или "needs_review"
  "checks":[
    {"mechanism":"qr_ttl","status":"passed","details":{...}},
    {"mechanism":"geo",   "status":"passed","details":{"distance_m":3.2,"radius_m":25}},
    {"mechanism":"wifi",  "status":"passed","details":{"expected_bssids":[...],"actual_bssid":"..."}}
  ]
}
```

Ошибки:
- `400 invalid_qr_token` — HMAC/формат
- `409 already_submitted` — студент уже отметился на этой сессии
- `409 session_not_accepting` — сессия закрыта / вне времени

## Teacher-override отметки (teacher + admin)

`PATCH /api/v1/attendance/:id` — ручной перевод записи в `accepted` / `rejected`.
Применяется к записям, которые автомат пометил `needs_review`. Teacher может
override'ить только отметки своих сессий; admin — любые.

```bash
curl -X PATCH $API/api/v1/attendance/<aid> -H "Authorization: Bearer $TEACHER" \
    -d '{"final_status":"accepted","notes":"подошёл лично, подтвердил присутствие"}'
```

Ответ: `AttendanceResponse` с заполненными `final_status`, `resolved_by`,
`resolved_at`, `notes`, `effective_status`.

Ошибки:
- `400 invalid_final_status` — не `accepted` / `rejected`
- `403 forbidden` — teacher пытается трогать чужую сессию
- `404 attendance_not_found`
- `409 not_resolvable` — уже resolved

После успешного override в WS-канал сессии отправляется сообщение
`{"type":"attendance_resolved","attendance_id":"<uuid>","final_status":"accepted","effective_status":"accepted"}`.

## WebSocket (teacher + admin)

`ws://localhost:8080/ws/sessions/<sid>/teacher`

**JWT передаётся через `Sec-WebSocket-Protocol: bearer.<jwt>`** — browser API не
позволяет поставить `Authorization` header на upgrade.

Клиент получает сообщения каждые `ttl_seconds`:

```json
{
  "type":"qr_token",
  "session_id":"<sid>",
  "counter":17,
  "token":"ZWUA-...",
  "expires_at":"2026-04-20T10:05:22Z"
}
```

И — после каждого submit'а студента:
```json
{
  "type":"attendance",
  "attendance_id":"<uuid>",
  "student_id":"<uuid>",
  "preliminary_status":"needs_review",
  "checks":[...]
}
```

Dev-утилита для ручной проверки:
```bash
go run ./cmd/wstest --session $SID --jwt $TEACHER --n 3
```

## Отчёты (teacher + admin)

Формат выбирается URL-расширением: `.xlsx` или `.csv`. Один из фильтров обязателен.

```bash
# Отчёт по сессии
curl -s -o session.xlsx "$API/api/v1/reports/attendance.xlsx?session_id=<sid>" \
    -H "Authorization: Bearer $TEACHER"

# По курсу за период
curl -s -o course.csv "$API/api/v1/reports/attendance.csv?course_id=<cid>&from=2026-04-01T00:00:00Z&to=2026-04-30T23:59:59Z" \
    -H "Authorization: Bearer $ADMIN"

# По группе
curl -s -o group.xlsx "$API/api/v1/reports/attendance.xlsx?group_id=<gid>" \
    -H "Authorization: Bearer $ADMIN"
```

Teacher **автоматически** ограничен своими сессиями — запрос по чужому course_id
вернёт пустой файл (только заголовки).

CSV идёт с BOM + разделителем `;` для совместимости с Excel на Windows.
XLSX — с заголовками жирным и подогнанной шириной колонок.

## Self-service студента

```bash
# Свои отметки с пагинацией
curl "$API/api/v1/students/me/attendance?limit=50&offset=0" \
    -H "Authorization: Bearer $STUDENT"

# Агрегат
curl "$API/api/v1/students/me/stats" -H "Authorization: Bearer $STUDENT"
# {"total":42,"accepted":40,"needs_review":2,"rejected":0,"attendance_rate":0.9523}
```

## Аудит (admin-only)

```bash
# Последние 50 событий
curl "$API/api/v1/audit?limit=50" -H "Authorization: Bearer $ADMIN"

# Фильтр по actor
curl "$API/api/v1/audit?actor_id=<uuid>&action=attendance_submitted" -H "Authorization: Bearer $ADMIN"

# Проверка целостности цепочки
curl -X POST $API/api/v1/audit/verify -H "Authorization: Bearer $ADMIN"
# {"ok":true,"total_entries":123}

# После ручной порчи в БД
# UPDATE audit_log SET payload = payload || '{"hack":true}'::jsonb WHERE id=50;
curl -X POST $API/api/v1/audit/verify -H "Authorization: Bearer $ADMIN"
# {"ok":false,"total_entries":49,"first_broken_id":50,"broken_reason":"record_hash mismatch"}
```

## CORS (для SPA)

API отвечает CORS-заголовками, если Origin запроса присутствует в
`CORS_ALLOWED_ORIGINS` (`.env`). По умолчанию разрешены два фронта:

```
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:5174
```

Preflight (OPTIONS) возвращает `204` и кешируется на 24 часа.
