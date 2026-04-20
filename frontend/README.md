# Frontend

Два SPA на Vite + React + TypeScript + Mantine.

| SPA | Путь | Порт | Назначение |
|---|---|---|---|
| Teacher | `frontend/teacher` | `:5173` | Преподавательский кабинет: сессии, live-QR, override отметок |
| Admin | `frontend/admin` | `:5174` | CRUD пользователей/политик/каталога, просмотр и verify аудита |

## Запуск

Оба SPA предполагают, что бэкенд поднят на `http://localhost:8080` (см. `backend/docker-compose.yml`).

```bash
# Teacher
cd frontend/teacher
npm install
npm run dev            # http://localhost:5173

# Admin
cd frontend/admin
npm install
npm run dev            # http://localhost:5174
```

`.env` в каждом SPA задаёт `VITE_API_BASE` (по умолчанию — localhost:8080) и, для teacher, `VITE_WS_BASE`.

## Стек

- **Vite 8** + **TypeScript** (strict).
- **Mantine 9** (`@mantine/core`, `@mantine/hooks`, `@mantine/notifications`, `@mantine/form`, `@mantine/dates`).
- **@tanstack/react-query 5** — кэш и мутации.
- **Zustand** — минималистичный store для auth (access/refresh токены + principal). Всё остальное — кэш react-query.
- **react-router 7** — роутинг.
- **react-qr-code** (только teacher) — рендер QR.

Без axios: `src/api/client.ts` — тонкая обёртка поверх `fetch`, единый refresh-on-401, typed `ApiError.code` для `switch`.

## Структура SPA

```
src/
├── api/
│   ├── client.ts       fetch + refresh + ApiError
│   ├── endpoints.ts    per-resource модули (authApi, sessionsApi, …)
│   └── types.ts        зеркало backend DTO, синхронизируется руками
├── auth/store.ts       Zustand persist
├── components/         AppShell, ProtectedRoute, ErrorBoundary, переиспользуемое
├── hooks/              (teacher) useTeacherSocket, useFullscreen
├── lib/                format, notify, clipboard
├── pages/              экраны
├── App.tsx             Routes
└── main.tsx            Mantine + QueryClient + Router + ErrorBoundary
```

## Почему не shared-package

Teacher и admin делят ~300 строк кода (apiClient, auth store, format-утилиты).
На курсовой shared package через workspaces добавил бы build-complexity
без практического выигрыша — поэтому намеренно дублируем.
