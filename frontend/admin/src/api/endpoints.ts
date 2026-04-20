import { apiRequest } from './client';
import type {
    AuditEntry,
    AuditVerifyResponse,
    Classroom,
    Course,
    CreateUserRequest,
    CreateUserResponse,
    Group,
    ListEnvelope,
    MeResponse,
    MechanismsConfig,
    Policy,
    Role,
    Stream,
    TokenResponse,
    UpdateUserRequest,
    User,
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

// ----- Users -----
export type UserListFilter = {
    role?: Role;
    q?: string;
    group_id?: string;
    limit?: number;
    offset?: number;
};

export const usersApi = {
    list: (filter: UserListFilter = {}) =>
        apiRequest<ListEnvelope<User>>('/users', { query: filter }),
    get: (id: string) => apiRequest<User>(`/users/${id}`),
    create: (body: CreateUserRequest) =>
        apiRequest<CreateUserResponse>('/users', { method: 'POST', body }),
    update: (id: string, body: UpdateUserRequest) =>
        apiRequest<User>(`/users/${id}`, { method: 'PATCH', body }),
    delete: (id: string) => apiRequest<void>(`/users/${id}`, { method: 'DELETE' }),
    resetPassword: (id: string) =>
        apiRequest<{ temp_password: string }>(`/users/${id}/reset-password`, { method: 'POST' }),
};

// ----- Policies -----
export type CreatePolicyRequest = {
    name: string;
    mechanisms: MechanismsConfig;
    is_default?: boolean;
};

export type UpdatePolicyRequest = {
    name?: string;
    mechanisms?: MechanismsConfig;
};

export const policiesApi = {
    list: () => apiRequest<ListEnvelope<Policy>>('/policies'),
    get: (id: string) => apiRequest<Policy>(`/policies/${id}`),
    create: (body: CreatePolicyRequest) =>
        apiRequest<Policy>('/policies', { method: 'POST', body }),
    update: (id: string, body: UpdatePolicyRequest) =>
        apiRequest<Policy>(`/policies/${id}`, { method: 'PATCH', body }),
    delete: (id: string) => apiRequest<void>(`/policies/${id}`, { method: 'DELETE' }),
    setDefault: (id: string) =>
        apiRequest<Policy>(`/policies/${id}/set-default`, { method: 'POST' }),
};

// ----- Catalog: courses -----
export const coursesApi = {
    list: () => apiRequest<ListEnvelope<Course>>('/courses'),
    get: (id: string) => apiRequest<Course>(`/courses/${id}`),
    create: (body: { name: string; code: string }) =>
        apiRequest<Course>('/courses', { method: 'POST', body }),
    update: (id: string, body: { name?: string; code?: string }) =>
        apiRequest<Course>(`/courses/${id}`, { method: 'PATCH', body }),
    delete: (id: string) => apiRequest<void>(`/courses/${id}`, { method: 'DELETE' }),
};

// ----- Catalog: groups -----
export const groupsApi = {
    list: () => apiRequest<ListEnvelope<Group>>('/groups'),
    create: (body: { name: string }) => apiRequest<Group>('/groups', { method: 'POST', body }),
    update: (id: string, body: { name?: string }) =>
        apiRequest<Group>(`/groups/${id}`, { method: 'PATCH', body }),
    delete: (id: string) => apiRequest<void>(`/groups/${id}`, { method: 'DELETE' }),
};

// ----- Catalog: streams -----
export const streamsApi = {
    listForCourse: (courseId: string) =>
        apiRequest<ListEnvelope<Stream>>('/streams', { query: { course_id: courseId } }),
    get: (id: string) => apiRequest<Stream>(`/streams/${id}`),
    create: (body: { course_id: string; name: string; group_ids: string[] }) =>
        apiRequest<Stream>('/streams', { method: 'POST', body }),
    update: (id: string, body: { name?: string; group_ids?: string[] }) =>
        apiRequest<Stream>(`/streams/${id}`, { method: 'PATCH', body }),
    delete: (id: string) => apiRequest<void>(`/streams/${id}`, { method: 'DELETE' }),
};

// ----- Catalog: classrooms -----
export const classroomsApi = {
    list: () => apiRequest<ListEnvelope<Classroom>>('/classrooms'),
    get: (id: string) => apiRequest<Classroom>(`/classrooms/${id}`),
    create: (body: {
        building: string;
        room_number: string;
        latitude: number;
        longitude: number;
        radius_m: number;
        allowed_bssids: string[];
    }) => apiRequest<Classroom>('/classrooms', { method: 'POST', body }),
    update: (
        id: string,
        body: Partial<{
            building: string;
            room_number: string;
            latitude: number;
            longitude: number;
            radius_m: number;
            allowed_bssids: string[];
        }>,
    ) => apiRequest<Classroom>(`/classrooms/${id}`, { method: 'PATCH', body }),
    delete: (id: string) => apiRequest<void>(`/classrooms/${id}`, { method: 'DELETE' }),
};

// ----- Audit -----
export type AuditListFilter = {
    action?: string;
    actor_id?: string;
    entity_type?: string;
    entity_id?: string;
    from?: string;
    to?: string;
    limit?: number;
    offset?: number;
};

export const auditApi = {
    list: (filter: AuditListFilter = {}) =>
        apiRequest<ListEnvelope<AuditEntry>>('/audit', { query: filter }),
    verify: () => apiRequest<AuditVerifyResponse>('/audit/verify', { method: 'POST' }),
};
