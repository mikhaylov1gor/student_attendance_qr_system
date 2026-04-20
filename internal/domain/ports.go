// Package domain содержит инфраструктурные порты, которые не привязаны
// к конкретной доменной сущности (криптопримитивы, часы).
//
// Сущности и репозитории живут в подпакетах (user, catalog, session, attendance,
// policy, audit).
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PasswordHasher — порт хеширования и проверки паролей. Реализация — argon2id
// (OWASP 2024 baseline), см. internal/infrastructure/crypto/argon2id.go.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

// FieldEncryptor — порт прозрачного шифрования отдельных полей (например, ФИО).
// Реализация — AES-256-GCM с ключом из env и 12-байтовым nonce из crypto/rand.
// Nonce уникален на каждую запись и хранится отдельной колонкой.
type FieldEncryptor interface {
	Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error)
	Decrypt(ciphertext, nonce []byte) ([]byte, error)
}

// QRToken — распарсенная полезная нагрузка QR-токена.
// Сам кодек (HMAC-SHA256 + base64url) живёт в infrastructure/crypto.
type QRToken struct {
	SessionID uuid.UUID
	Counter   int
	IssuedAt  time.Time
}

// QRTokenCodec — порт кодирования/декодирования подписанных QR-токенов.
//
// Encode строит токен из полезной нагрузки и qr_secret сессии;
// Decode проверяет подпись (constant-time) и возвращает payload или ошибку.
// Проверка свежести counter выполняется на уровне сервиса (Submit), не здесь.
type QRTokenCodec interface {
	Encode(secret []byte, payload QRToken) (string, error)
	Decode(secret []byte, token string) (QRToken, error)
}

// Clock — порт получения текущего времени. Прокидывается везде, где время
// влияет на бизнес-логику (TTL, аудит, ротация). В тестах подменяется на
// фиксированные часы.
type Clock interface {
	Now(ctx context.Context) time.Time
}

// TxRunner — порт управления транзакциями. Реализация (infrastructure/db)
// открывает транзакцию, кладёт её в ctx и вызывает fn. Репозитории внутри fn
// через хелпер dbx(ctx) автоматически используют эту транзакцию.
//
// Используется, когда нужно атомарно выполнить несколько операций из разных
// use case'ов/репозиториев — в первую очередь для audit append + основная
// мутация в одной транзакции.
type TxRunner interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}

// RotatorController — порт управления QR-ротатором.
// Реализация (application/qr) поднимает per-session goroutine, которая
// инкрементирует counter и публикует QR через Hub в WebSocket-каналы.
//
// Вызовы — side effect жизненного цикла сессии: SessionService.Start →
// rotator.Start, SessionService.Close → rotator.Stop. На bootstrap'е
// приложения Manager сам поднимает rotator'ы для всех status=active.
type RotatorController interface {
	Start(sessionID uuid.UUID, secret []byte, ttlSeconds int) error
	Stop(sessionID uuid.UUID) error
}
