package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditLogModel — tamper-evident журнал действий.
// ID — bigserial, источник порядка для hash-chain.
//
// IPAddress хранится как *string: колонка inet, но Gorm через pgx не умеет
// сканировать её в net.IP (driver.Value string → *net.IP unsupported).
// Преобразование в net.IP — в mapper domain↔model.
type AuditLogModel struct {
	ID         int64      `gorm:"column:id;primaryKey;autoIncrement"`
	PrevHash   []byte     `gorm:"column:prev_hash;type:bytea;not null"`
	RecordHash []byte     `gorm:"column:record_hash;type:bytea;not null"`
	OccurredAt time.Time  `gorm:"type:timestamptz;not null;default:now();index"`
	ActorID    *uuid.UUID `gorm:"type:uuid;index"`
	ActorRole  string     `gorm:"type:text"`
	Action     string     `gorm:"type:text;not null"`
	EntityType string     `gorm:"type:text;not null"`
	EntityID   string     `gorm:"type:text;not null"`
	Payload    JSONB      `gorm:"type:jsonb;not null"`
	IPAddress  *string    `gorm:"column:ip_address;type:inet"`
	UserAgent  string     `gorm:"type:text"`
}

func (AuditLogModel) TableName() string { return "audit_log" }
