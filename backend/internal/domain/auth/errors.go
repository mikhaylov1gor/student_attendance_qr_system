package auth

import "errors"

var (
	// ErrInvalidCredentials — неверная пара email/password.
	// Отдельная сентинел-ошибка, чтобы handler мог отмаппить в 401.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrInvalidToken — подпись не валидна или payload испорчен.
	ErrInvalidToken = errors.New("auth: invalid token")
	// ErrTokenExpired — токен протух по exp/expires_at.
	ErrTokenExpired = errors.New("auth: token expired")
	// ErrTokenRevoked — refresh-токен уже отозван (logout или ротация).
	ErrTokenRevoked = errors.New("auth: token revoked")
	// ErrForbidden — у пользователя нет нужной роли/прав.
	ErrForbidden = errors.New("auth: forbidden")
	// ErrUnauthorized — нет Principal в контексте.
	ErrUnauthorized = errors.New("auth: unauthorized")
)
