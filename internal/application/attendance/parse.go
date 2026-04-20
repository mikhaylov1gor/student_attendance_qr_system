package attendance

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain"
)

// parseSessionFromToken читает только session_id / counter / issued_at из
// base64url-токена БЕЗ верификации HMAC. Используется для loadSession — чтобы
// узнать, из какой сессии тянуть qr_secret, и уже ПОСЛЕ проверить подпись.
//
// Формат — тот же, что в infrastructure/crypto/qrtoken.go:
//
//	[0..16) session_id | [16..24) counter | [24..32) issued_at_ns | [32..64) hmac32
func parseSessionFromToken(token string) (domain.QRToken, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return domain.QRToken{}, err
	}
	if len(raw) != 64 {
		return domain.QRToken{}, errors.New("qrtoken: bad length")
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
