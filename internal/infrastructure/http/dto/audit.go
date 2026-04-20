package dto

import (
	"encoding/hex"
	"time"

	"attendance/internal/domain/audit"
)

// AuditEntryResponse — одна запись журнала для API.
type AuditEntryResponse struct {
	ID         int64          `json:"id"`
	PrevHash   string         `json:"prev_hash"`   // hex
	RecordHash string         `json:"record_hash"` // hex
	OccurredAt time.Time      `json:"occurred_at"`
	ActorID    *string        `json:"actor_id,omitempty"`
	ActorRole  string         `json:"actor_role,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Payload    map[string]any `json:"payload,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
}

func AuditEntryFromDomain(e audit.Entry) AuditEntryResponse {
	var actorID *string
	if e.ActorID != nil {
		s := e.ActorID.String()
		actorID = &s
	}
	var ip string
	if e.IPAddress != nil {
		ip = e.IPAddress.String()
	}
	return AuditEntryResponse{
		ID:         e.ID,
		PrevHash:   hex.EncodeToString(e.PrevHash),
		RecordHash: hex.EncodeToString(e.RecordHash),
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

// AuditListResponse — GET /audit.
type AuditListResponse struct {
	Items []AuditEntryResponse `json:"items"`
	Total int                  `json:"total"`
}

// AuditVerifyResponse — POST /audit/verify.
type AuditVerifyResponse struct {
	OK            bool   `json:"ok"`
	TotalEntries  int    `json:"total_entries"`
	FirstBrokenID *int64 `json:"first_broken_id,omitempty"`
	BrokenReason  string `json:"broken_reason,omitempty"`
}
