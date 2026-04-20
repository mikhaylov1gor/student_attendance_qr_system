package audit

import (
	"crypto/sha256"
	"fmt"
)

// GenesisPrevHash — prev_hash для самой первой (genesis) записи в цепочке.
// Все нули, длиной HashLen.
func GenesisPrevHash() []byte {
	return make([]byte, HashLen)
}

// ComputeRecordHash вычисляет record_hash для новой записи цепочки.
//
//	record_hash = SHA256(prev_hash ‖ Canonicalize(event))
//
// prev_hash должен быть ровно HashLen байт; Canonicalize() выполняется внутри,
// чтобы исключить сдвиг из-за разной реализации у писателя и читателя.
func ComputeRecordHash(prevHash []byte, e Entry) ([]byte, error) {
	if len(prevHash) != HashLen {
		return nil, fmt.Errorf("audit: prev_hash must be %d bytes, got %d", HashLen, len(prevHash))
	}
	canon, err := Canonicalize(e.ToCanonical())
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write(prevHash)
	h.Write(canon)
	return h.Sum(nil), nil
}

// ToCanonical извлекает из Entry поля, покрываемые цепочкой, и нормализует
// представление (uuid → string, net.IP → string). Не используется в БД-mapper'е,
// только для детерминированного хэширования.
func (e Entry) ToCanonical() CanonicalEvent {
	var actorID string
	if e.ActorID != nil {
		actorID = e.ActorID.String()
	}
	var ip string
	if e.IPAddress != nil {
		ip = e.IPAddress.String()
	}
	return CanonicalEvent{
		OccurredAt: e.OccurredAt,
		ActorID:    actorID,
		ActorRole:  e.ActorRole,
		Action:     string(e.Action),
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Payload:    e.Payload,
		IPAddress:  ip,
		UserAgent:  e.UserAgent,
	}
}
