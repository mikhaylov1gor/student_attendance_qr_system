package models

import (
	"time"

	"github.com/google/uuid"
)

// UserModel — Gorm-модель users.
// FullNameCiphertext + FullNameNonce — AES-256-GCM; plaintext сущность
// пересобирается на mapper-уровне через FieldEncryptor.
type UserModel struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email              string     `gorm:"type:text;uniqueIndex;not null"`
	PasswordHash       string     `gorm:"type:text;not null"`
	FullNameCiphertext []byte     `gorm:"type:bytea;not null"`
	FullNameNonce      []byte     `gorm:"type:bytea;not null"`
	Role               string     `gorm:"type:user_role;not null"`
	CurrentGroupID     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt          *time.Time `gorm:"type:timestamptz;index"`
}

func (UserModel) TableName() string { return "users" }
