package auth

import (
	"context"

	"github.com/google/uuid"
)

// RefreshTokenRepository — персистентный слой для refresh-токенов.
// Plaintext токен никогда не передаётся в репозиторий — только его SHA-256.
type RefreshTokenRepository interface {
	// Store сохраняет новую запись о выданном refresh-токене.
	Store(ctx context.Context, t RefreshToken) error

	// GetByHash ищет активную (не отозванную) запись по хэшу plaintext'а.
	// Если не найдена — ErrInvalidToken.
	// Если отозвана/протухла — запись возвращается, but caller обязан вызвать
	// .IsActive() и вернуть соответствующую доменную ошибку.
	GetByHash(ctx context.Context, tokenHash []byte) (RefreshToken, error)

	// Revoke помечает запись revoked_at = now(). Повторный вызов — no-op.
	Revoke(ctx context.Context, id uuid.UUID) error

	// RevokeAllForUser отзывает все ещё активные токены пользователя (используется
	// при смене пароля, компрометации, или ручном kick'е).
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// AccessTokenSigner — порт подписания/проверки access-JWT.
type AccessTokenSigner interface {
	Sign(claims AccessClaims) (string, error)
	Verify(token string) (AccessClaims, error)
}
