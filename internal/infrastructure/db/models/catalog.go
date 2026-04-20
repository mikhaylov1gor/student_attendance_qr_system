package models

import (
	"time"

	"github.com/google/uuid"
)

// ==== courses ====

type CourseModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string     `gorm:"type:text;not null"`
	Code      string     `gorm:"type:text;uniqueIndex;not null"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt *time.Time `gorm:"type:timestamptz"`
}

func (CourseModel) TableName() string { return "courses" }

// ==== groups ====

type GroupModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string     `gorm:"type:text;uniqueIndex;not null"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt *time.Time `gorm:"type:timestamptz"`
}

func (GroupModel) TableName() string { return "groups" }

// ==== streams + stream_groups ====

type StreamModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CourseID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	Name      string     `gorm:"type:text;not null"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt *time.Time `gorm:"type:timestamptz"`
}

func (StreamModel) TableName() string { return "streams" }

type StreamGroupModel struct {
	StreamID uuid.UUID `gorm:"type:uuid;primaryKey"`
	GroupID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (StreamGroupModel) TableName() string { return "stream_groups" }

// ==== classrooms ====

type ClassroomModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Building      string     `gorm:"type:text;not null"`
	RoomNumber    string     `gorm:"type:text;not null"`
	Latitude      float64    `gorm:"type:double precision;not null"`
	Longitude     float64    `gorm:"type:double precision;not null"`
	RadiusM       int        `gorm:"column:radius_m;type:integer;not null"`
	AllowedBSSIDs JSONB      `gorm:"column:allowed_bssids;type:jsonb;not null;default:'[]'::jsonb"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt     *time.Time `gorm:"type:timestamptz"`
}

func (ClassroomModel) TableName() string { return "classrooms" }
