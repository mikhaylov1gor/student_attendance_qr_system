package models

import (
	"time"

	"github.com/google/uuid"
)

type SecurityPolicyModel struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name       string     `gorm:"type:text;uniqueIndex;not null"`
	Mechanisms JSONB      `gorm:"type:jsonb;not null"`
	IsDefault  bool       `gorm:"type:boolean;not null;default:false"`
	CreatedBy  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt  *time.Time `gorm:"type:timestamptz"`
}

func (SecurityPolicyModel) TableName() string { return "security_policies" }
