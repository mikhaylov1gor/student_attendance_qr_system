import { type ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';

import { useAuthStore } from '../auth/store';

export function ProtectedRoute({ children }: { children: ReactNode }) {
    const principal = useAuthStore((s) => s.principal);
    const token = useAuthStore((s) => s.accessToken);
    const location = useLocation();

    if (!token || !principal) {
        const returnTo = encodeURIComponent(location.pathname + location.search);
        return <Navigate to={`/login?return_to=${returnTo}`} replace />;
    }

    // Teacher SPA: разрешены teacher + admin (админ тоже может вести сессии).
    if (principal.role !== 'teacher' && principal.role !== 'admin') {
        return <Navigate to="/login" replace />;
    }

    return <>{children}</>;
}
