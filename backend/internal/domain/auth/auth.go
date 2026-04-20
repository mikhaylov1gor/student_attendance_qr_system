// Package auth описывает доменные типы аутентификации и порты.
// Use case'ы живут в internal/application/auth.
package auth

import (
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain/user"
)

// Principal — идентификация пользователя внутри запроса.
// Кладётся в context.Context auth-middleware'ом, читается хендлерами и
// сервисами (для audit-append).
type Principal struct {
	UserID uuid.UUID
	Role   user.Role
}

// AccessClaims — claims access-JWT (HS256).
//
// sub — user_id, role — роль (для быстрого guard'а в middleware без обращения к БД),
// jti — уникальный id токена (полезен для audit).
type AccessClaims struct {
	Subject  uuid.UUID
	Role     user.Role
	IssuedAt time.Time
	Expires  time.Time
	TokenID  uuid.UUID // jti
}

// TokenPair — результат Login и Refresh.
type TokenPair struct {
	AccessToken    string
	RefreshToken   string
	ExpiresIn      time.Duration // для access; клиенту показываем в секундах
	AccessExpires  time.Time
	RefreshExpires time.Time
}

// RefreshToken — persisted refresh-token. Сам plaintext токен никогда не
// хранится, только TokenHash = SHA-256(plaintext).
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte // 32 байта, SHA-256
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// IsActive — не отозван и не истёк.
func (t RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
