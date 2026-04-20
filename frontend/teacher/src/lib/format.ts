const dateTimeFormatter = new Intl.DateTimeFormat('ru-RU', {
    dateStyle: 'medium',
    timeStyle: 'short',
});

const timeFormatter = new Intl.DateTimeFormat('ru-RU', { timeStyle: 'medium' });

export function formatDateTime(iso: string | Date): string {
    const d = typeof iso === 'string' ? new Date(iso) : iso;
    return dateTimeFormatter.format(d);
}

export function formatTime(iso: string | Date): string {
    const d = typeof iso === 'string' ? new Date(iso) : iso;
    return timeFormatter.format(d);
}
