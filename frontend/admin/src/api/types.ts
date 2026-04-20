// API-типы admin SPA. Зеркалят backend DTO; синхронизировать вручную.

export type Role = 'admin' | 'teacher' | 'student';

// ----- Auth -----
export type TokenResponse = {
    access_token: string;
    refresh_token: string;
    expires_in: number;
    expires_at: string;
    token_type: 'Bearer';
};

export type MeResponse = {
    id: string;
    email: string;
    role: Role;
    full_name: string;
    current_group_id?: string;
};

// ----- Users -----
export type User = {
    id: string;
    email: string;
    full_name: string;
    role: Role;
    current_group_id?: string;
    created_at: string;
};

export type CreateUserRequest = {
    email: string;
    password?: string;
    role: Role;
    last: string;
    first: string;
    middle?: string;
    current_group_id?: string;
};

export type CreateUserResponse = {
    user: User;
    temp_password?: string;
};

export type UpdateUserRequest = {
    email?: string;
    role?: Role;
    last?: string;
    first?: string;
    middle?: string;
    current_group_id?: string;
    clear_group?: boolean;
};

// ----- Catalog -----
export type Course = { id: string; name: string; code: string; created_at: string };
export type Group = { id: string; name: string; created_at: string };
export type Stream = {
    id: string;
    course_id: string;
    name: string;
    group_ids: string[];
    created_at: string;
};
export type Classroom = {
    id: string;
    building: string;
    room_number: string;
    latitude: number;
    longitude: number;
    radius_m: number;
    allowed_bssids: string[];
    created_at: string;
};

export type ListEnvelope<T> = { items: T[]; total: number };

// ----- Policies -----
export type MechanismsConfig = {
    qr_ttl: { enabled: boolean; ttl_seconds: number };
    geo: { enabled: boolean; radius_override_m?: number };
    wifi: {
        enabled: boolean;
        required_bssids_from_classroom?: boolean;
        extra_bssids?: string[];
    };
    bluetooth_beacon: { enabled: boolean };
};

export type Policy = {
    id: string;
    name: string;
    mechanisms: MechanismsConfig;
    is_default: boolean;
    created_at: string;
};

// ----- Audit -----
export type AuditEntry = {
    id: number;
    prev_hash: string;
    record_hash: string;
    occurred_at: string;
    actor_id?: string;
    actor_role?: string;
    action: string;
    entity_type: string;
    entity_id: string;
    payload?: Record<string, unknown>;
    ip_address?: string;
    user_agent?: string;
};

export type AuditVerifyResponse = {
    ok: boolean;
    total_entries: number;
    first_broken_id?: number;
    broken_reason?: string;
};
