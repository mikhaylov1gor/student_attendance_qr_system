package attendance

import (
	"context"

	"github.com/google/uuid"
)

// ListFilter — параметры выборки отметок.
type ListFilter struct {
	SessionID *uuid.UUID
	StudentID *uuid.UUID
	FromTime  *string
	ToTime    *string
	Limit     int
	Offset    int
}

// Repository — порт репозитория отметок.
//
// Submit записывает пару (Record + []CheckResult) в одной транзакции.
// Транзакционность — деталь инфраструктуры; здесь мы лишь фиксируем контракт.
type Repository interface {
	Submit(ctx context.Context, r Record, checks []CheckResult) error
	Resolve(ctx context.Context, id uuid.UUID, finalStatus Status, resolvedBy uuid.UUID, notes string) error

	GetByID(ctx context.Context, id uuid.UUID) (Record, []CheckResult, error)
	ExistsForSessionStudent(ctx context.Context, sessionID, studentID uuid.UUID) (bool, error)

	List(ctx context.Context, f ListFilter) ([]Record, int, error)
}
