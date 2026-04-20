// Package crypto — реализации крипто-портов domain (PasswordHasher,
// FieldEncryptor) и низкоуровневые помощники (HMAC-SHA256).
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idParams — настраиваемые параметры.
// Дефолты задаются в config и соответствуют OWASP 2024 baseline.
type Argon2idParams struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

// DefaultArgon2idParams — OWASP 2024 (m=64MB, t=3, p=2, salt=16, key=32).
func DefaultArgon2idParams() Argon2idParams {
	return Argon2idParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLen:     16,
		KeyLen:      32,
	}
}

// Argon2idHasher реализует domain.PasswordHasher.
type Argon2idHasher struct {
	params Argon2idParams
}

func NewArgon2idHasher(p Argon2idParams) *Argon2idHasher {
	if p.SaltLen == 0 {
		p.SaltLen = 16
	}
	if p.KeyLen == 0 {
		p.KeyLen = 32
	}
	return &Argon2idHasher{params: p}
}

// Hash возвращает self-describing PHC-строку вида:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<saltB64>$<hashB64>
//
// Verify читает параметры из самой строки, поэтому смена параметров не ломает
// существующие хэши.
func (h *Argon2idHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2id: read salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLen,
	)

	b64 := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory, h.params.Iterations, h.params.Parallelism,
		b64.EncodeToString(salt),
		b64.EncodeToString(key),
	), nil
}

// Verify возвращает true, если password соответствует encodedHash.
// Сравнение constant-time (subtle.ConstantTimeCompare).
func (h *Argon2idHasher) Verify(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	// "" $argon2id $v=19 $m=...,t=...,p=... $salt $hash  → 6 частей
	if len(parts) != 6 {
		return false, errPHCFormat
	}
	if parts[1] != "argon2id" {
		return false, errPHCFormat
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, errPHCFormat
	}
	if version != argon2.Version {
		return false, fmt.Errorf("argon2id: unsupported version %d", version)
	}

	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, errPHCFormat
	}

	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errPHCFormat
	}
	expected, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, errPHCFormat
	}

	actual := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(expected)))
	return subtle.ConstantTimeCompare(expected, actual) == 1, nil
}

var errPHCFormat = errors.New("argon2id: invalid PHC-encoded hash")
