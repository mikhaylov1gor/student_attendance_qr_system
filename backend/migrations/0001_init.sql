-- Migration 0001: init — полная схема БД для системы учёта посещаемости.
-- Соответствует docs/architecture/er-model.md.

-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- для gen_random_uuid()

-- ====================================================================
-- ENUM-типы
-- ====================================================================

CREATE TYPE user_role         AS ENUM ('student', 'teacher', 'admin');
CREATE TYPE session_status    AS ENUM ('draft', 'active', 'closed');
CREATE TYPE attendance_status AS ENUM ('accepted', 'needs_review', 'rejected');
CREATE TYPE check_status      AS ENUM ('passed', 'failed', 'skipped');

-- ====================================================================
-- Таблицы справочников: courses, groups, streams, stream_groups, classrooms
-- ====================================================================

CREATE TABLE courses (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    code       text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE groups (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE streams (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id  uuid        NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (course_id, name)
);
CREATE INDEX idx_streams_course_id ON streams(course_id);

CREATE TABLE stream_groups (
    stream_id uuid NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    group_id  uuid NOT NULL REFERENCES groups(id)  ON DELETE RESTRICT,
    PRIMARY KEY (stream_id, group_id)
);
CREATE INDEX idx_stream_groups_group_id ON stream_groups(group_id);

CREATE TABLE classrooms (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    building        text        NOT NULL,
    room_number     text        NOT NULL,
    latitude        double precision NOT NULL,
    longitude       double precision NOT NULL,
    radius_m        integer     NOT NULL CHECK (radius_m > 0),
    allowed_bssids  jsonb       NOT NULL DEFAULT '[]'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz,
    UNIQUE (building, room_number)
);
CREATE INDEX idx_classrooms_allowed_bssids ON classrooms USING GIN (allowed_bssids);

-- ====================================================================
-- users
-- ====================================================================

CREATE TABLE users (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email                text        NOT NULL UNIQUE,
    password_hash        text        NOT NULL,          -- $argon2id$v=19$...
    full_name_ciphertext bytea       NOT NULL,          -- AES-256-GCM
    full_name_nonce      bytea       NOT NULL,          -- 12 байт, уникальный на запись
    role                 user_role   NOT NULL,
    current_group_id     uuid        REFERENCES groups(id) ON DELETE RESTRICT,
    created_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz,

    CONSTRAINT users_nonce_len_check    CHECK (octet_length(full_name_nonce) = 12),
    CONSTRAINT users_role_group_check   CHECK (role = 'student' OR current_group_id IS NULL)
);
CREATE INDEX idx_users_role             ON users(role);
CREATE INDEX idx_users_current_group_id ON users(current_group_id);

-- ====================================================================
-- security_policies
-- ====================================================================

CREATE TABLE security_policies (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL UNIQUE,
    mechanisms jsonb       NOT NULL,
    is_default boolean     NOT NULL DEFAULT false,
    created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

-- Ровно одна активная default-политика на БД.
CREATE UNIQUE INDEX uniq_default_policy
    ON security_policies (is_default)
    WHERE is_default = true AND deleted_at IS NULL;

-- ====================================================================
-- sessions
-- ====================================================================

CREATE TABLE sessions (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id         uuid        NOT NULL REFERENCES users(id)             ON DELETE RESTRICT,
    course_id          uuid        NOT NULL REFERENCES courses(id)           ON DELETE RESTRICT,
    classroom_id       uuid        REFERENCES classrooms(id)                 ON DELETE RESTRICT,
    security_policy_id uuid        NOT NULL REFERENCES security_policies(id) ON DELETE RESTRICT,
    starts_at          timestamptz NOT NULL,
    ends_at            timestamptz NOT NULL,
    status             session_status NOT NULL DEFAULT 'draft',
    qr_secret          bytea       NOT NULL,  -- 32 байта, ephemeral
    qr_ttl_seconds     integer     NOT NULL CHECK (qr_ttl_seconds BETWEEN 3 AND 120),
    qr_counter         integer     NOT NULL DEFAULT 0,
    created_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT sessions_time_order_check CHECK (ends_at > starts_at),
    CONSTRAINT sessions_qr_secret_len_check CHECK (octet_length(qr_secret) = 32)
);
CREATE INDEX idx_sessions_teacher_id ON sessions(teacher_id);
CREATE INDEX idx_sessions_course_id  ON sessions(course_id);
CREATE INDEX idx_sessions_starts_at  ON sessions(starts_at);
CREATE INDEX idx_sessions_status     ON sessions(status);

CREATE TABLE session_groups (
    session_id uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    group_id   uuid NOT NULL REFERENCES groups(id)   ON DELETE RESTRICT,
    PRIMARY KEY (session_id, group_id)
);
CREATE INDEX idx_session_groups_group_id ON session_groups(group_id);

-- ====================================================================
-- attendance_records + security_check_results
-- ====================================================================

CREATE TABLE attendance_records (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id           uuid              NOT NULL REFERENCES sessions(id) ON DELETE RESTRICT,
    student_id           uuid              NOT NULL REFERENCES users(id)    ON DELETE RESTRICT,
    submitted_at         timestamptz       NOT NULL DEFAULT now(),
    submitted_qr_token   text              NOT NULL,
    preliminary_status   attendance_status NOT NULL,
    final_status         attendance_status,
    resolved_by          uuid              REFERENCES users(id) ON DELETE SET NULL,
    resolved_at          timestamptz,
    notes                text,

    CONSTRAINT attendance_final_status_check
        CHECK (final_status IS NULL OR final_status IN ('accepted', 'rejected')),
    CONSTRAINT attendance_resolved_consistency_check
        CHECK ((final_status IS NULL AND resolved_by IS NULL AND resolved_at IS NULL)
            OR (final_status IS NOT NULL AND resolved_by IS NOT NULL AND resolved_at IS NOT NULL))
);

-- Ключевой инвариант: один студент — одна отметка на сессию.
CREATE UNIQUE INDEX uniq_attendance_session_student
    ON attendance_records (session_id, student_id);

CREATE INDEX idx_attendance_session_id ON attendance_records(session_id);
CREATE INDEX idx_attendance_student_id ON attendance_records(student_id);
CREATE INDEX idx_attendance_submitted_at ON attendance_records(submitted_at);

CREATE TABLE security_check_results (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    attendance_id uuid         NOT NULL REFERENCES attendance_records(id) ON DELETE CASCADE,
    mechanism     text         NOT NULL,           -- qr_ttl | geo | wifi | ...
    status        check_status NOT NULL,
    details       jsonb        NOT NULL DEFAULT '{}'::jsonb,
    checked_at    timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX idx_check_results_attendance_id ON security_check_results(attendance_id);

-- ====================================================================
-- audit_log (hash-chain)
-- ====================================================================

CREATE TABLE audit_log (
    id          bigserial PRIMARY KEY,
    prev_hash   bytea       NOT NULL,          -- 32 байта; для genesis — все нули
    record_hash bytea       NOT NULL,          -- SHA-256(prev_hash ‖ canonical_payload)
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_id    uuid        REFERENCES users(id) ON DELETE SET NULL,
    actor_role  text,
    action      text        NOT NULL,
    entity_type text        NOT NULL,
    entity_id   text        NOT NULL,
    payload     jsonb       NOT NULL,
    ip_address  inet,
    user_agent  text,

    CONSTRAINT audit_prev_hash_len_check   CHECK (octet_length(prev_hash)   = 32),
    CONSTRAINT audit_record_hash_len_check CHECK (octet_length(record_hash) = 32)
);
CREATE INDEX idx_audit_log_occurred_at ON audit_log(occurred_at);
CREATE INDEX idx_audit_log_actor_id    ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action      ON audit_log(action);
CREATE INDEX idx_audit_log_entity      ON audit_log(entity_type, entity_id);

-- +goose Down

DROP TABLE IF EXISTS audit_log              CASCADE;
DROP TABLE IF EXISTS security_check_results CASCADE;
DROP TABLE IF EXISTS attendance_records     CASCADE;
DROP TABLE IF EXISTS session_groups         CASCADE;
DROP TABLE IF EXISTS sessions               CASCADE;
DROP TABLE IF EXISTS security_policies      CASCADE;
DROP TABLE IF EXISTS users                  CASCADE;
DROP TABLE IF EXISTS classrooms             CASCADE;
DROP TABLE IF EXISTS stream_groups          CASCADE;
DROP TABLE IF EXISTS streams                CASCADE;
DROP TABLE IF EXISTS groups                 CASCADE;
DROP TABLE IF EXISTS courses                CASCADE;

DROP TYPE IF EXISTS check_status;
DROP TYPE IF EXISTS attendance_status;
DROP TYPE IF EXISTS session_status;
DROP TYPE IF EXISTS user_role;
