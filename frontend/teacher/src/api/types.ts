// Типы API — отзеркаливают backend DTO. Держим их руками (openapi codegen в плане
// сознательно не делаем), поэтому при изменении контракта синхронизировать здесь.

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

// ----- Session -----
export type SessionStatus = 'draft' | 'active' | 'closed';

export type Session = {
    id: string;
    teacher_id: string;
    course_id: string;
    classroom_id?: string;
    security_policy_id: string;
    starts_at: string;
    ends_at: string;
    status: SessionStatus;
    qr_ttl_seconds: number;
    qr_counter: number;
    group_ids: string[];
    created_at: string;
};

export type SessionList = { items: Session[]; total: number };

export type CreateSessionRequest = {
    course_id: string;
    classroom_id?: string | null;
    security_policy_id?: string | null;
    starts_at: string;
    ends_at: string;
    group_ids: string[];
    qr_ttl_seconds?: number;
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

// ----- Attendance -----
export type AttendanceStatus = 'accepted' | 'needs_review' | 'rejected';

export type CheckResult = {
    mechanism: string;
    status: 'passed' | 'warning' | 'failed' | string;
    details?: Record<string, unknown>;
    checked_at: string;
};

export type AttendanceRecord = {
    id: string;
    session_id: string;
    student_id: string;
    submitted_at: string;
    preliminary_status: AttendanceStatus;
    final_status?: AttendanceStatus;
    resolved_by?: string;
    resolved_at?: string;
    notes?: string;
    effective_status: AttendanceStatus;
    checks: CheckResult[];
};

export type ResolveAttendanceRequest = {
    final_status: 'accepted' | 'rejected';
    notes?: string;
};

// ----- Policy -----
export type Policy = {
    id: string;
    name: string;
    mechanisms: MechanismsConfig;
    is_default: boolean;
    created_at: string;
};

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

// ----- WebSocket messages (teacher channel) -----
export type WsQRMessage = {
    type: 'qr_token';
    session_id: string;
    counter: number;
    token: string;
    expires_at: string;
};

export type WsAttendanceMessage = {
    type: 'attendance';
    attendance_id: string;
    session_id: string;
    student_id: string;
    submitted_at: string;
    preliminary_status: AttendanceStatus;
    checks: Array<{ mechanism: string; status: string; details?: Record<string, unknown> }>;
};

export type WsAttendanceResolvedMessage = {
    type: 'attendance_resolved';
    attendance_id: string;
    session_id: string;
    student_id: string;
    final_status: 'accepted' | 'rejected';
    effective_status: AttendanceStatus;
};

export type WsMessage = WsQRMessage | WsAttendanceMessage | WsAttendanceResolvedMessage;
