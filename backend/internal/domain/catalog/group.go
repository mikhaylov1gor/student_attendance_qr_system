package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Group — академическая группа (например, БПИ-221). Стабильна, не привязана
// к курсу; состав студентов определяется через User.CurrentGroupID.
type Group struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	DeletedAt *time.Time
}
