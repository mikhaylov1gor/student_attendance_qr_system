import { notifications } from '@mantine/notifications';

import { ApiError } from '../api/client';

export function notifyApiError(e: unknown, title = 'Ошибка') {
    const msg = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e);
    notifications.show({ color: 'red', title, message: msg });
}

export function notifySuccess(message: string) {
    notifications.show({ color: 'green', message });
}
