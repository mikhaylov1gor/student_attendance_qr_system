// Package catalog содержит справочные сущности: курсы, группы, потоки и
// аудитории. Эти сущности живут относительно независимо и редко изменяются,
// поэтому сгруппированы в один пакет.
package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Course — учебный курс (дисциплина). Code — короткий машинный идентификатор
// (например, CS-BD), уникальный.
type Course struct {
	ID        uuid.UUID
	Name      string
	Code      string
	CreatedAt time.Time
	DeletedAt *time.Time
}
