import { apiRequest } from './client';
import type {
    AttendanceRecord,
    Classroom,
    Course,
    CreateSessionRequest,
    Group,
    ListEnvelope,
    MeResponse,
    ResolveAttendanceRequest,
    Session,
    SessionList,
    SessionStatus,
    Stream,
    TokenResponse,
} from './types';

// ----- Auth -----
export const authApi = {
    login: (email: string, password: string) =>
        apiRequest<TokenResponse>('/auth/login', {
            method: 'POST',
            body: { email, password },
        }),
    logout: (refreshToken: string) =>
        apiRequest<void>('/auth/logout', {
            method: 'POST',
            body: { refresh_token: refreshToken },
        }),
    me: () => apiRequest<MeResponse>('/auth/me'),
};

// ----- Sessions -----
export type SessionListFilter = {
    teacher_id?: string;
    course_id?: string;
    status?: SessionStatus;
    from?: string;
    to?: string;
    limit?: number;
    offset?: number;
};

export const sessionsApi = {
    list: (filter: SessionListFilter = {}) =>
        apiRequest<SessionList>('/sessions', { query: filter }),
    get: (id: string) => apiRequest<Session>(`/sessions/${id}`),
    create: (body: CreateSessionRequest) =>
        apiRequest<Session>('/sessions', { method: 'POST', body }),
    update: (id: string, body: Partial<CreateSessionRequest> & { clear_classroom?: boolean }) =>
        apiRequest<Session>(`/sessions/${id}`, { method: 'PATCH', body }),
    start: (id: string) => apiRequest<Session>(`/sessions/${id}/start`, { method: 'POST' }),
    close: (id: string) => apiRequest<Session>(`/sessions/${id}/close`, { method: 'POST' }),
    delete: (id: string) => apiRequest<void>(`/sessions/${id}`, { method: 'DELETE' }),
};

// ----- Attendance -----
export const attendanceApi = {
    resolve: (id: string, body: ResolveAttendanceRequest) =>
        apiRequest<AttendanceRecord>(`/attendance/${id}`, { method: 'PATCH', body }),
};

// ----- Catalog -----
export const catalogApi = {
    courses: {
        list: () => apiRequest<ListEnvelope<Course>>('/courses'),
        get: (id: string) => apiRequest<Course>(`/courses/${id}`),
    },
    groups: {
        list: () => apiRequest<ListEnvelope<Group>>('/groups'),
    },
    streams: {
        listForCourse: (courseId: string) =>
            apiRequest<ListEnvelope<Stream>>('/streams', { query: { course_id: courseId } }),
    },
    classrooms: {
        list: () => apiRequest<ListEnvelope<Classroom>>('/classrooms'),
    },
};

// ----- Reports -----
export const reportsApi = {
    downloadSessionXlsx: async (sessionId: string): Promise<Blob> => {
        const res = await apiRequest<Response>(`/reports/attendance.xlsx`, {
            query: { session_id: sessionId },
            raw: true,
        });
        return res.blob();
    },
};
