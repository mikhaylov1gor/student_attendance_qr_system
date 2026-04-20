# HTTP API — эндпоинты

Первичный черновик REST API. Все эндпоинты под префиксом `/api/v1`. Все, кроме `POST /auth/login` и `POST /auth/refresh`, требуют заголовок `Authorization: Bearer <access_token>`.

## Конвенции

- **Формат обмена:** JSON (UTF-8).
- **Коды ошибок:** 400 — невалидные данные, 401 — нет/истёк access-токен, 403 — недостаточно прав для роли, 404 — ресурс не найден, 409 — нарушение инварианта (например, группа не принадлежит курсу), 422 — семантическая ошибка, 500 — сервер.
- **Формат ошибки:** `{ "error": { "code": "string", "message": "string", "details": {...} } }`.
- **Пагинация:** query-параметры `?page=1&per_page=50`, ответ содержит `{items, meta: {page, per_page, total}}`.
- **Сортировка/фильтрация:** query-параметры, описываются в конкретных эндпоинтах.
- **Timestamp'ы:** ISO-8601 UTC, с суффиксом `Z`.

## Аутентификация

| Метод | Путь                  | Роль   | Описание                                            |
|-------|-----------------------|--------|-----------------------------------------------------|
| POST  | `/auth/login`         | все    | `{email, password}` → `{access_token, refresh_token, user}` |
| POST  | `/auth/refresh`       | все    | `{refresh_token}` → `{access_token, refresh_token}` (ротация refresh) |
| POST  | `/auth/logout`        | все    | Инвалидация refresh-токена на сервере               |
| GET   | `/auth/me`            | все    | Профиль текущего пользователя                       |

## Администратор

### Пользователи
| Метод  | Путь              | Описание                                      |
|--------|-------------------|-----------------------------------------------|
| GET    | `/users`          | Фильтры: `?role=`, `?group_id=`, `?search=` (поиск по email, по ФИО — после расшифровки в памяти) |
| POST   | `/users`          | Создать пользователя с временным паролем     |
| GET    | `/users/:id`      |                                               |
| PATCH  | `/users/:id`      | Изменить ФИО, роль, `current_group_id`        |
| DELETE | `/users/:id`      | Soft delete (`deleted_at`)                    |
| POST   | `/users/:id/reset-password` | Сгенерировать временный пароль      |

### Справочники: группы, потоки, курсы, аудитории
| Метод  | Путь                                         | Описание                       |
|--------|----------------------------------------------|--------------------------------|
| GET    | `/groups`                                    | Список с фильтром `?search=`   |
| POST   | `/groups`                                    | `{name}`                       |
| GET    | `/groups/:id`                                |                                |
| PATCH  | `/groups/:id`                                |                                |
| DELETE | `/groups/:id`                                | Запрет, если есть студенты     |
| GET    | `/streams`                                   | Обязательный `?course_id=`     |
| POST   | `/streams`                                   | `{course_id, name}`            |
| PATCH  | `/streams/:id`                               |                                |
| DELETE | `/streams/:id`                               |                                |
| POST   | `/streams/:id/groups`                        | `{group_id}` — добавить группу в поток |
| DELETE | `/streams/:id/groups/:group_id`              | Убрать группу из потока        |
| GET    | `/courses`                                   |                                |
| POST   | `/courses`                                   | `{name, code}`                 |
| PATCH  | `/courses/:id`                               |                                |
| DELETE | `/courses/:id`                               |                                |
| GET    | `/classrooms`                                |                                |
| POST   | `/classrooms`                                | `{building, room_number, latitude, longitude, radius_m, allowed_bssids}` |
| PATCH  | `/classrooms/:id`                            |                                |
| DELETE | `/classrooms/:id`                            |                                |

### Политики защиты
| Метод | Путь                                    | Описание                                      |
|-------|-----------------------------------------|-----------------------------------------------|
| GET   | `/security-policies`                    | Список всех политик                           |
| POST  | `/security-policies`                    | `{name, mechanisms: {...}}` (см. ниже структуру `mechanisms`) |
| GET   | `/security-policies/:id`                |                                               |
| PATCH | `/security-policies/:id`                | Правка конфигурации механизмов                |
| DELETE| `/security-policies/:id`                | Soft delete, запрет удаления default           |
| POST  | `/security-policies/:id/set-default`    | Сделать эту политику default (транзакционно снимает флаг с предыдущей) |

Формат `mechanisms`:
```json
{
  "qr_ttl":  { "enabled": true, "ttl_seconds": 10 },
  "geo":     { "enabled": true, "radius_override_m": null },
  "wifi":    { "enabled": true, "required_bssids_from_classroom": true }
}
```

### Аудит
| Метод | Путь                  | Описание                                                    |
|-------|-----------------------|-------------------------------------------------------------|
| GET   | `/audit`              | Фильтры: `?action=`, `?actor_id=`, `?entity_type=`, `?from=`, `?to=` + пагинация |
| POST  | `/audit/verify`       | Полная верификация hash-chain; возвращает `{ok: bool, first_broken_id?: int}` |

## Преподаватель

### Сессии
| Метод  | Путь                                              | Описание                                                                                     |
|--------|---------------------------------------------------|----------------------------------------------------------------------------------------------|
| GET    | `/sessions`                                       | Фильтры: `?mine=true`, `?course_id=`, `?status=`, `?from=`, `?to=`                           |
| POST   | `/sessions`                                       | `{course_id, classroom_id?, security_policy_id, group_ids: [], starts_at, ends_at, qr_ttl_override?}` |
| GET    | `/sessions/:id`                                   | Полная информация о сессии                                                                    |
| PATCH  | `/sessions/:id`                                   | Только в статусе `draft` — редактирование                                                     |
| POST   | `/sessions/:id/start`                             | `draft → active`, инициирует генерацию `qr_secret`                                            |
| POST   | `/sessions/:id/close`                             | `active → closed`                                                                             |
| DELETE | `/sessions/:id`                                   | Только в статусе `draft`                                                                      |

### Отметки в сессии
| Метод | Путь                                                     | Описание                                                     |
|-------|----------------------------------------------------------|--------------------------------------------------------------|
| GET   | `/sessions/:id/attendance`                               | Список студентов сессии + результаты проверок для каждого    |
| PATCH | `/sessions/:id/attendance/:attendance_id`                | `{final_status: "accepted"\|"rejected", notes?}` — ручное решение преподавателя |

### Отчёты (синхронные)
| Метод | Путь                                          | Описание                                                       |
|-------|-----------------------------------------------|----------------------------------------------------------------|
| GET   | `/reports/attendance.xlsx`                    | Параметры: `?course_id=`, `?group_id=` / `?stream_id=`, `?from=`, `?to=` — отдаёт Excel-файл потоком |
| GET   | `/reports/attendance.csv`                     | То же, формат CSV                                              |

`Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` или `text/csv; charset=utf-8`. `Content-Disposition: attachment; filename="..."`.

## Студент (мобильное приложение)

| Метод | Путь                                          | Описание                                                                                                          |
|-------|-----------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| POST  | `/attendance`                                 | `{qr_token, geo: {lat, lng, accuracy}, wifi: {bssid, ssid}?, client_time}` → `{attendance_id, preliminary_status, checks: [...]}` |
| GET   | `/students/me/attendance`                     | Фильтры: `?from=`, `?to=`, `?course_id=` + пагинация                                                              |
| GET   | `/students/me/stats`                          | Агрегированная статистика: по курсу, за период                                                                     |

**Важно:** `POST /attendance` **всегда возвращает 200**, даже если проверки не прошли. Результат проверок — в теле ответа, в массиве `checks`. Это следствие политики non-blocking (см. [project_security_mechanisms_policy](../../../.claude/projects/C--Users-mikha-Documents-kursovaya/memory/project_security_mechanisms_policy.md)).

## WebSocket

| Путь                                            | Роль         | Назначение                                                                    |
|-------------------------------------------------|--------------|-------------------------------------------------------------------------------|
| `/ws/sessions/:id/teacher`                      | преподаватель| Канал преподавателя на время активной сессии                                  |

**Аутентификация:** JWT access-токен передаётся в query-параметре `?token=` или через subprotocol `access_token.<jwt>` (второе предпочтительнее, т.к. не попадает в access-log).

**Сообщения от сервера:**
- `{"type": "qr_token", "token": "<base64>", "counter": 42, "expires_at": "..."}` — каждые `qr_ttl_seconds` секунд.
- `{"type": "attendance", "record": {...}}` — при новой отметке в сессии.
- `{"type": "session_closed"}` — при закрытии сессии, сервер инициирует close.

**Сообщения от клиента:** в MVP не предусмотрены (канал однонаправленный). Client→Server команды могут быть добавлены позже (pause/resume ротации).

## Матрица доступа по ролям

| Группа эндпоинтов          | student | teacher | admin |
|----------------------------|---------|---------|-------|
| `/auth/*`                  | ✓       | ✓       | ✓     |
| `/users/*`                 | —       | —       | ✓     |
| `/groups/*`, `/streams/*`, `/courses/*`, `/classrooms/*` | — | GET (для UI создания сессии) | ✓ |
| `/security-policies/*`     | —       | GET     | ✓ (все) |
| `/audit/*`                 | —       | —       | ✓     |
| `/sessions/*`              | —       | ✓       | GET   |
| `/reports/*`               | —       | ✓       | ✓     |
| `POST /attendance`         | ✓       | —       | —     |
| `/students/me/*`           | ✓       | —       | —     |
| `/ws/sessions/*/teacher`   | —       | ✓       | —     |

## Что осознанно не моделируется в API MVP

- **Пакетная загрузка студентов в группу** (CSV-импорт) — добавится при необходимости.
- **Уведомления студентам** (email/push об отметке) — вне scope.
- **Изменение студентом профиля** — не предусмотрено, админ управляет.
- **Revocation конкретного QR-токена** — следствие stateless-схемы, короткий TTL компенсирует.
- **Async-отчёты** (task queue с polling статуса) — осознанно отклонено как преждевременная оптимизация.
