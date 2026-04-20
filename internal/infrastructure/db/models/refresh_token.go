package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshTokenModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash []byte     `gorm:"type:bytea;not null;uniqueIndex"`
	IssuedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null;index"`
	RevokedAt *time.Time `gorm:"type:timestamptz"`
}

func (RefreshTokenModel) TableName() string { return "refresh_tokens" }
