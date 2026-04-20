import { useAuthStore } from '../auth/store';

export const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080/api/v1';
export const WS_BASE = import.meta.env.VITE_WS_BASE ?? 'ws://localhost:8080/ws';

// ApiError — typed exception. Backend возвращает {"error":{"code","message"}},
// код стабильный → компоненты свитчатся по `.code`, не парся message.
export class ApiError extends Error {
    code: string;
    httpStatus: number;
    details?: unknown;

    constructor(code: string, httpStatus: number, message: string, details?: unknown) {
        super(message);
        this.name = 'ApiError';
        this.code = code;
        this.httpStatus = httpStatus;
        this.details = details;
    }
}

type RefreshResponse = {
    access_token: string;
    refresh_token: string;
    expires_in: number;
};

// Единственный in-flight refresh, чтобы параллельные 401-ответы не запускали
// дубли (при ротации любой повторный рефреш второго токена сразу провалится).
let refreshInFlight: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
    if (refreshInFlight) return refreshInFlight;
    const refreshToken = useAuthStore.getState().refreshToken;
    if (!refreshToken) return false;

    refreshInFlight = (async () => {
        try {
            const res = await fetch(`${API_BASE}/auth/refresh`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ refresh_token: refreshToken }),
            });
            if (!res.ok) {
                useAuthStore.getState().clear();
                return false;
            }
            const data: RefreshResponse = await res.json();
            useAuthStore.getState().setSession({
                accessToken: data.access_token,
                refreshToken: data.refresh_token,
                principal: useAuthStore.getState().principal!,
            });
            return true;
        } catch {
            useAuthStore.getState().clear();
            return false;
        } finally {
            refreshInFlight = null;
        }
    })();
    return refreshInFlight;
}

export type RequestOptions = {
    method?: string;
    body?: unknown;
    query?: Record<string, string | number | boolean | undefined | null>;
    headers?: Record<string, string>;
    signal?: AbortSignal;
    // raw: true — вернуть Response (нужно для скачивания xlsx и подобного).
    raw?: boolean;
};

function buildUrl(path: string, query?: RequestOptions['query']) {
    const url = new URL(API_BASE + path);
    if (query) {
        for (const [k, v] of Object.entries(query)) {
            if (v === undefined || v === null) continue;
            url.searchParams.set(k, String(v));
        }
    }
    return url.toString();
}

export async function apiRequest<T = unknown>(path: string, opts: RequestOptions = {}): Promise<T> {
    const run = async (): Promise<Response> => {
        const token = useAuthStore.getState().accessToken;
        const headers: Record<string, string> = {
            Accept: 'application/json',
            ...opts.headers,
        };
        if (opts.body !== undefined && !(opts.body instanceof FormData)) {
            headers['Content-Type'] = 'application/json';
        }
        if (token) headers['Authorization'] = `Bearer ${token}`;

        return fetch(buildUrl(path, opts.query), {
            method: opts.method ?? 'GET',
            headers,
            body:
                opts.body === undefined
                    ? undefined
                    : opts.body instanceof FormData
                      ? opts.body
                      : JSON.stringify(opts.body),
            signal: opts.signal,
        });
    };

    let res = await run();
    if (res.status === 401) {
        const ok = await tryRefresh();
        if (ok) {
            res = await run();
        }
    }

    if (!res.ok) {
        let code = 'http_error';
        let message = res.statusText;
        let details: unknown;
        try {
            const data = await res.json();
            if (data?.error) {
                code = data.error.code ?? code;
                message = data.error.message ?? message;
                details = data.error.details;
            }
        } catch {
            // not json — оставим httpStatus/statusText.
        }
        throw new ApiError(code, res.status, message, details);
    }

    if (opts.raw) return res as unknown as T;
    if (res.status === 204) return undefined as T;
    const ctype = res.headers.get('Content-Type') ?? '';
    if (!ctype.includes('application/json')) return undefined as T;
    return (await res.json()) as T;
}
