package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
)

// HMACSHA256 — тонкая обёртка поверх stdlib.
// Используется кодеком QR-токена (этап 9).

// Sign возвращает HMAC-SHA256(key, data) как сырые 32 байта.
func HMACSHA256Sign(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// HMACSHA256Verify сравнивает переданный тег с пересчитанным в constant-time.
func HMACSHA256Verify(key, data, tag []byte) bool {
	expected := HMACSHA256Sign(key, data)
	return subtle.ConstantTimeCompare(expected, tag) == 1
}
