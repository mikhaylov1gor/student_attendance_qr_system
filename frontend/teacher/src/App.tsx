import { Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from './components/AppShell';
import { ProtectedRoute } from './components/ProtectedRoute';
import { LoginPage } from './pages/LoginPage';
import { SessionCreatePage } from './pages/SessionCreatePage';
import { SessionDetailsPage } from './pages/SessionDetailsPage';
import { SessionLivePage } from './pages/SessionLivePage';
import { SessionsListPage } from './pages/SessionsListPage';

export function App() {
    return (
        <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
                element={
                    <ProtectedRoute>
                        <AppShell />
                    </ProtectedRoute>
                }
            >
                <Route index element={<Navigate to="/sessions" replace />} />
                <Route path="/sessions" element={<SessionsListPage />} />
                <Route path="/sessions/new" element={<SessionCreatePage />} />
                <Route path="/sessions/:id" element={<SessionDetailsPage />} />
            </Route>

            {/* Live-экран — без AppShell, полноэкранный QR. */}
            <Route
                path="/sessions/:id/live"
                element={
                    <ProtectedRoute>
                        <SessionLivePage />
                    </ProtectedRoute>
                }
            />

            <Route path="*" element={<Navigate to="/sessions" replace />} />
        </Routes>
    );
}
