package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Stream — поток в контексте конкретного курса. «БПИ-21+22 на мат.анализе» и
// «БПИ-21+22 на философии» — разные Stream даже при совпадающем составе групп.
// Состав задаётся через M:N stream_groups.
type Stream struct {
	ID        uuid.UUID
	CourseID  uuid.UUID
	Name      string
	GroupIDs  []uuid.UUID // состав потока
	CreatedAt time.Time
	DeletedAt *time.Time
}
