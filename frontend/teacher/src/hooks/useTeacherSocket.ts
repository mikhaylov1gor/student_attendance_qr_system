import { useEffect, useRef, useState } from 'react';

import { WS_BASE } from '../api/client';
import type { WsMessage } from '../api/types';
import { useAuthStore } from '../auth/store';

type Status = 'connecting' | 'open' | 'closed' | 'error';

// useTeacherSocket — устанавливает WebSocket к /ws/sessions/:id/teacher,
// передаёт JWT через Sec-WebSocket-Protocol (browser API не даёт ставить
// Authorization на upgrade). Реконнект с exponential backoff.
export function useTeacherSocket(sessionId: string | undefined) {
    const [status, setStatus] = useState<Status>('connecting');
    const [messages, setMessages] = useState<WsMessage[]>([]);
    const retryRef = useRef(0);
    const wsRef = useRef<WebSocket | null>(null);

    useEffect(() => {
        if (!sessionId) return;

        let closed = false;
        let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

        const connect = () => {
            const token = useAuthStore.getState().accessToken;
            if (!token) {
                setStatus('error');
                return;
            }
            setStatus('connecting');
            const url = `${WS_BASE}/sessions/${sessionId}/teacher`;
            const ws = new WebSocket(url, [`bearer.${token}`]);
            wsRef.current = ws;

            ws.onopen = () => {
                retryRef.current = 0;
                setStatus('open');
            };
            ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data) as WsMessage;
                    setMessages((prev) => [...prev, data]);
                } catch {
                    // Пропускаем невалидный JSON — лог не захламляем.
                }
            };
            ws.onerror = () => setStatus('error');
            ws.onclose = () => {
                if (closed) return;
                setStatus('closed');
                const attempt = retryRef.current++;
                const delay = Math.min(1000 * 2 ** attempt, 15_000);
                reconnectTimer = setTimeout(connect, delay);
            };
        };

        connect();

        return () => {
            closed = true;
            if (reconnectTimer) clearTimeout(reconnectTimer);
            wsRef.current?.close();
        };
    }, [sessionId]);

    return { status, messages };
}
