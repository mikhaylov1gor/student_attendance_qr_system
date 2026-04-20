package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Classroom — учебная аудитория с предустановленными координатами и списком
// разрешённых Wi-Fi BSSID. Админ ведёт справочник; преподаватель при создании
// сессии просто выбирает аудиторию.
type Classroom struct {
	ID            uuid.UUID
	Building      string
	RoomNumber    string
	Latitude      float64
	Longitude     float64
	RadiusMeters  int
	AllowedBSSIDs []string
	CreatedAt     time.Time
	DeletedAt     *time.Time
}
