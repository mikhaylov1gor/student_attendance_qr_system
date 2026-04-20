import { useCallback, useEffect, useRef, useState } from 'react';

// Обёртка вокруг Fullscreen API. Возвращает ref для целевого элемента,
// текущий флаг fullscreen и toggle-функцию.
export function useFullscreen<T extends HTMLElement = HTMLDivElement>() {
    const ref = useRef<T | null>(null);
    const [fullscreen, setFullscreen] = useState(false);

    useEffect(() => {
        const onChange = () => setFullscreen(document.fullscreenElement !== null);
        document.addEventListener('fullscreenchange', onChange);
        return () => document.removeEventListener('fullscreenchange', onChange);
    }, []);

    const toggle = useCallback(async () => {
        if (document.fullscreenElement) {
            await document.exitFullscreen().catch(() => {});
            return;
        }
        if (ref.current) {
            await ref.current.requestFullscreen().catch(() => {});
        }
    }, []);

    return { ref, fullscreen, toggle };
}
