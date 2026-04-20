# План фронта

Документ фиксирует решения, принятые **до** начала разработки, чтобы в новой
сессии Claude стартовал с известного брифа, а не выспрашивал заново.

## Структура монорепы

```
kursovaya/
├── backend/              Go API (готов, см. backend/docs/)
├── frontend/
│   ├── teacher/          SPA преподавателя
│   └── admin/            SPA админа
└── docs/
    └── frontend-plan.md  ← этот файл
```

Мобилка (React Native + Expo) — в отдельной сессии после фронта.

## Контракт с бэкендом

- Базовый URL dev: `http://localhost:8080/api/v1`
- WebSocket: `ws://localhost:8080/ws/sessions/:id/teacher`, JWT через `Sec-WebSocket-Protocol: bearer.<jwt>`
- Авторизация: `Authorization: Bearer <access_token>`; refresh — через `/auth/refresh` с ротацией (старый refresh сразу отзывается).
- Формат ошибок: `{"error":{"code":"<stable_code>","message":"<human>"}}`. Коды — стабильные, на них можно делать `switch`.
- CORS уже включён для `http://localhost:5173` (teacher) и `http://localhost:5174` (admin) — настроено в `backend/.env.example` (`CORS_ALLOWED_ORIGINS`).
- Rate-limit на `/auth/login` — 10/мин/IP → `429 rate_limited` + `Retry-After`.

Полный каталог endpoint'ов: [../backend/docs/api-examples.md](../backend/docs/api-examples.md).

## Стек (общий для обеих SPA)

| Слой | Выбор | Комментарий |
|---|---|---|
| Билдер | **Vite** | fast HMR, нативный TS |
| Язык | **TypeScript** (strict) | типизированные API-клиенты |
| UI | **Mantine 7** (`@mantine/core`, `@mantine/hooks`, `@mantine/notifications`, `@mantine/form`) | готовые компоненты, тёмная тема коробочно |
| Роутер | **react-router v6** | стандарт |
| Data | **@tanstack/react-query v5** | cache, refetch, optimistic updates |
| Формы | **@mantine/form + zod** | валидация + типы из одной схемы |
| Global state | **Zustand** | только для auth (principal + tokens) — остальное кэш react-query |
| HTTP | `fetch` + тонкая обёртка (`apiClient.ts`) | без axios, экономим bundle |

### Обоснование некоторых решений

- **Mantine vs shadcn/ui:** Mantine даёт готовые компоненты из коробки, shadcn — «копируй-вставляй». На курсовой скорость важнее кастомизации.
- **react-query для data, Zustand для auth:** данные с бэка кэшируются/инвалидируются react-query'ем. Zustand держит JWT + profile, которые не «данные endpoint'а», а глобальная сессия.
- **Без axios:** fetch хватает; `apiClient.ts` добавляет Bearer автоматически, делает refresh на 401, маппит ошибку в typed exception.

## Teacher SPA (порт :5173)

### Роуты

```
/login                  публичный
/                       redirect → /sessions
/sessions               список моих сессий
/sessions/new           форма создания draft
/sessions/:id           детали сессии (draft/active/closed)
/sessions/:id/live      QR на весь экран + live-поток отметок (active only)
```

### Must-have экраны

1. **Login** — email/password, при успехе кладём токены в Zustand + localStorage.
2. **Sessions list** — свои сессии с фильтрами по status/course; быстрые actions: Start (draft), Close (active).
3. **Session create** — форма: course, classroom (опц), groups (multi-select из `GroupsForCourse`), starts/ends, QR TTL override. Обработка `409 groups_not_in_course_streams`.
4. **Session live** — на весь экран:
   - **QR-код** (через react-qr-code) из последнего WS-сообщения. Counter + expires_at в углу.
   - **Таблица отметок** — live через тот же WS, `{type:"attendance"}` добавляет строку.
   - Для записей с `preliminary_status=needs_review` — кнопки Accept/Reject (PATCH `/attendance/:id` с `final_status` — этот endpoint нужно добавить в бэкенд, ниже).
5. **Отчёт** — кнопка «Скачать xlsx» на сессии → `GET /reports/attendance.xlsx?session_id=...` с `Authorization`.

### Открытый вопрос — teacher-ручной override

Сейчас в бэкенде есть `attendance.Resolve(id, finalStatus, ...)` в репо, но **нет HTTP-ручки**. Перед стартом фронта надо добавить `PATCH /api/v1/attendance/:id` (teacher-only, только для сессий которые он ведёт). ~30 минут работы в бэке.

## Admin SPA (порт :5174)

### Роуты

```
/login                   публичный
/                        redirect → /users
/users                   список + поиск по ФИО (?q=...)
/users/new               форма создания (показывает temp_password после)
/users/:id               редактирование + reset-password
/policies                список политик
/policies/new            создание
/policies/:id            редактирование + set-default
/catalog/courses         CRUD курсов
/catalog/groups          CRUD групп
/catalog/streams         CRUD потоков (фильтр по course_id)
/catalog/classrooms      CRUD аудиторий (координаты + BSSID)
/audit                   журнал + кнопка «Verify chain»
```

### Must-have интеракции

1. **User create/reset-password:** модал с copyable temp-паролем (pointer-блокер на кнопку закрытия до первого копирования).
2. **Policy mechanisms form:** JSON-схема в виде Mantine-формы, с живым превью `ttl_seconds`.
3. **Policy `set-default`:** кнопка с конфирмом; после — инвалидация react-query кэша.
4. **Catalog classrooms:** карта (Leaflet?) — опционально. Коробочно — `latitude/longitude` input'ами.
5. **Audit verify** — кнопка `POST /audit/verify`. Если `ok:false` — большой красный блок с `first_broken_id` и `reason`. **Это должен быть эффектный экран для защиты** — можно добавить хэш-chain визуализацию (стрелочки между записями).

## Auth-flow на фронте (обе SPA)

```typescript
// apiClient.ts — единая точка
async function apiRequest(path, opts) {
    const res = await fetch(API_BASE + path, {
        ...opts,
        headers: {
            ...opts.headers,
            'Authorization': `Bearer ${useAuthStore.getState().accessToken}`,
            'Content-Type': 'application/json',
        },
    });
    if (res.status === 401) {
        // Попытка refresh
        const refreshed = await tryRefresh();
        if (refreshed) return apiRequest(path, opts); // retry once
        useAuthStore.getState().clear();
        navigate('/login');
    }
    if (!res.ok) {
        const err = await res.json();
        throw new ApiError(err.error.code, err.error.message, res.status);
    }
    return res.json();
}
```

`ApiError` даёт `.code` → компоненты делают `if (e.code === 'already_submitted') {...}` без парсинга сообщений.

## Порядок работы в следующей сессии

1. **Бэк-доработка:** добавить `PATCH /api/v1/attendance/:id` (teacher-only override).
2. **Teacher SPA scaffold:** `npm create vite@latest frontend/teacher -- --template react-ts`; install Mantine/react-query/react-router/zod.
3. **apiClient.ts + useAuth store + типы из API** (генерация руками, без openapi).
4. **Login + Sessions list + Create + Start/Close** (без live-экрана) — первая вертикаль end-to-end.
5. **Session live** (QR + WS) — это гвоздь программы.
6. **Admin SPA scaffold** аналогично, на :5174.
7. **Users CRUD + Policies CRUD** — большая часть admin'а.
8. **Catalog + Audit verify** — докручиваем.
9. **Polish:** shared UI layout (AppShell), error boundaries, notifications на всё важное.

## Что сознательно НЕ делаем

- **SSR / Next.js** — статичный SPA достаточно; SSR не упрощает ничего для аутентифицированного портала.
- **GraphQL / OpenAPI codegen** — REST endpoint'ов <50, руками быстрее и нагляднее для курсовой.
- **E2E-тесты (Playwright/Cypress)** — для объёма курсовой избыточно.
- **Design system из коробки** (Storybook + tokens) — Mantine покрывает.
- **i18n** — только русский.
- **PWA / offline** — только онлайн.

## Риски / подводные камни

| Риск | Митигация |
|---|---|
| WebSocket reconnect при сетевых сбоях | Коробочно в react-query нет — накидаем простой retry с exponential backoff в `useTeacherSocket` hook. |
| QR слишком маленький на проекторе | На `/sessions/:id/live` — `useFullscreen` хук; QR рендерим в 80vh через react-qr-code. |
| Два SPA на разных портах не ходят одновременно в API (CORS) | Уже учтено в `.env.example`, оба origin'а в whitelist'е. |
| Expired refresh → пустой экран | `apiClient` при 401 и неудачном refresh делает `navigate('/login')` с `?return_to=...`. |
| Teacher override над чужой сессией | Бэк уже возвращает 403; на фронте просто показываем notification. |
