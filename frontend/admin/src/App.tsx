import { Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from './components/AppShell';
import { ProtectedRoute } from './components/ProtectedRoute';
import { AuditPage } from './pages/AuditPage';
import { ClassroomsPage } from './pages/catalog/ClassroomsPage';
import { CoursesPage } from './pages/catalog/CoursesPage';
import { GroupsPage } from './pages/catalog/GroupsPage';
import { StreamsPage } from './pages/catalog/StreamsPage';
import { LoginPage } from './pages/LoginPage';
import { PolicyEditPage } from './pages/PolicyEditPage';
import { PolicyNewPage } from './pages/PolicyNewPage';
import { PoliciesListPage } from './pages/PoliciesListPage';
import { UserEditPage } from './pages/UserEditPage';
import { UserNewPage } from './pages/UserNewPage';
import { UsersListPage } from './pages/UsersListPage';

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
                <Route index element={<Navigate to="/users" replace />} />

                <Route path="/users" element={<UsersListPage />} />
                <Route path="/users/new" element={<UserNewPage />} />
                <Route path="/users/:id" element={<UserEditPage />} />

                <Route path="/policies" element={<PoliciesListPage />} />
                <Route path="/policies/new" element={<PolicyNewPage />} />
                <Route path="/policies/:id" element={<PolicyEditPage />} />

                <Route path="/catalog/courses" element={<CoursesPage />} />
                <Route path="/catalog/groups" element={<GroupsPage />} />
                <Route path="/catalog/streams" element={<StreamsPage />} />
                <Route path="/catalog/classrooms" element={<ClassroomsPage />} />

                <Route path="/audit" element={<AuditPage />} />
            </Route>

            <Route path="*" element={<Navigate to="/users" replace />} />
        </Routes>
    );
}
