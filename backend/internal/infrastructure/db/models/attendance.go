package models

import (
	"time"

	"github.com/google/uuid"
)

type AttendanceRecordModel struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	StudentID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	SubmittedAt       time.Time  `gorm:"type:timestamptz;not null;default:now();index"`
	SubmittedQRToken  string     `gorm:"type:text;not null"`
	PreliminaryStatus string     `gorm:"type:attendance_status;not null"`
	FinalStatus       *string    `gorm:"type:attendance_status"`
	ResolvedBy        *uuid.UUID `gorm:"type:uuid"`
	ResolvedAt        *time.Time `gorm:"type:timestamptz"`
	Notes             string     `gorm:"type:text"`
}

func (AttendanceRecordModel) TableName() string { return "attendance_records" }

type SecurityCheckResultModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AttendanceID uuid.UUID `gorm:"type:uuid;not null;index"`
	Mechanism    string    `gorm:"type:text;not null"`
	Status       string    `gorm:"type:check_status;not null"`
	Details      JSONB     `gorm:"type:jsonb;not null;default:'{}'::jsonb"`
	CheckedAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (SecurityCheckResultModel) TableName() string { return "security_check_results" }
