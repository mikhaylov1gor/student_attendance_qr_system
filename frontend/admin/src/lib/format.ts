const dateTimeFormatter = new Intl.DateTimeFormat('ru-RU', {
    dateStyle: 'medium',
    timeStyle: 'short',
});

export function formatDateTime(iso: string | Date): string {
    const d = typeof iso === 'string' ? new Date(iso) : iso;
    return dateTimeFormatter.format(d);
}

// Копирует текст в clipboard. Возвращает true при успехе.
export async function copyToClipboard(text: string): Promise<boolean> {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch {
        return false;
    }
}

// truncate hash по центру: abc123…def456.
export function shortHash(hex: string, head = 6, tail = 6): string {
    if (hex.length <= head + tail + 1) return hex;
    return `${hex.slice(0, head)}…${hex.slice(-tail)}`;
}
