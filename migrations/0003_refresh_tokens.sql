-- Migration 0003: refresh_tokens — таблица для хранения refresh-токенов.
--
-- Токен в открытом виде никогда не хранится. При issue кладём SHA-256(token).
-- При ротации старая запись помечается revoked_at; при Logout — то же самое.

-- +goose Up

CREATE TABLE refresh_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL,
    issued_at   timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,

    CONSTRAINT refresh_tokens_hash_len_check CHECK (octet_length(token_hash) = 32),
    CONSTRAINT refresh_tokens_time_order_check CHECK (expires_at > issued_at)
);

CREATE UNIQUE INDEX uniq_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_user_id      ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at   ON refresh_tokens(expires_at);

-- +goose Down

DROP TABLE IF EXISTS refresh_tokens CASCADE;
