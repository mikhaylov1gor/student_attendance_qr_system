package crypto

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain"
)

// QRTokenCodec реализует domain.QRTokenCodec.
//
// Формат полезной нагрузки (raw, до base64url):
//
//	[0..16)   session_id           — uuid (16 байт)
//	[16..24)  counter              — uint64 BigEndian
//	[24..32)  issued_at_unix_nano  — int64  BigEndian
//	[32..64)  hmac32               — HMAC-SHA256(secret, [0..32))
//
// Кодирование — base64url (без паддинга): 64 байта raw → 86 символов.
type QRTokenCodec struct{}

func NewQRTokenCodec() *QRTokenCodec { return &QRTokenCodec{} }

var _ domain.QRTokenCodec = (*QRTokenCodec)(nil)

const (
	qrTokenRawLen = 64
	qrTokenSigOff = 32
)

// Encode строит подписанный токен. secret должен быть равен 32 байтам
// (см. session.QRSecretLen и CHECK в миграции 0001).
func (QRTokenCodec) Encode(secret []byte, p domain.QRToken) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("qrtoken: empty secret")
	}
	buf := make([]byte, qrTokenRawLen)
	copy(buf[0:16], p.SessionID[:])
	binary.BigEndian.PutUint64(buf[16:24], uint64(p.Counter))
	binary.BigEndian.PutUint64(buf[24:32], uint64(p.IssuedAt.UTC().UnixNano()))

	tag := HMACSHA256Sign(secret, buf[:qrTokenSigOff])
	copy(buf[qrTokenSigOff:], tag)

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Decode проверяет подпись constant-time и возвращает payload или ошибку.
// Проверка свежести (counter ∈ slack, expiresAt) — задача QRTTLCheck в engine.
func (QRTokenCodec) Decode(secret []byte, token string) (domain.QRToken, error) {
	if len(secret) == 0 {
		return domain.QRToken{}, errors.New("qrtoken: empty secret")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return domain.QRToken{}, fmt.Errorf("qrtoken: base64: %w", err)
	}
	if len(raw) != qrTokenRawLen {
		return domain.QRToken{}, fmt.Errorf("qrtoken: bad length %d", len(raw))
	}
	if !HMACSHA256Verify(secret, raw[:qrTokenSigOff], raw[qrTokenSigOff:]) {
		return domain.QRToken{}, errors.New("qrtoken: invalid signature")
	}

	var sid uuid.UUID
	copy(sid[:], raw[0:16])
	counter := binary.BigEndian.Uint64(raw[16:24])
	issuedNs := int64(binary.BigEndian.Uint64(raw[24:32]))

	return domain.QRToken{
		SessionID: sid,
		Counter:   int(counter),
		IssuedAt:  time.Unix(0, issuedNs).UTC(),
	}, nil
}
