package models

import (
	"time"

	"github.com/google/uuid"
)

type SessionModel struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TeacherID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	CourseID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	ClassroomID      *uuid.UUID `gorm:"type:uuid"`
	SecurityPolicyID uuid.UUID  `gorm:"type:uuid;not null"`
	StartsAt         time.Time  `gorm:"type:timestamptz;not null;index"`
	EndsAt           time.Time  `gorm:"type:timestamptz;not null"`
	Status           string     `gorm:"type:session_status;not null;default:'draft'"`
	QRSecret         []byte     `gorm:"column:qr_secret;type:bytea;not null"`
	QRTTLSeconds     int        `gorm:"column:qr_ttl_seconds;type:integer;not null"`
	QRCounter        int        `gorm:"column:qr_counter;type:integer;not null;default:0"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (SessionModel) TableName() string { return "sessions" }

type SessionGroupModel struct {
	SessionID uuid.UUID `gorm:"type:uuid;primaryKey"`
	GroupID   uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (SessionGroupModel) TableName() string { return "session_groups" }
