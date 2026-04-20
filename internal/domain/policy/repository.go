package policy

import (
	"context"

	"github.com/google/uuid"
)

// Repository — CRUD над SecurityPolicy и управление default-флагом.
type Repository interface {
	Create(ctx context.Context, p SecurityPolicy) error
	Update(ctx context.Context, p SecurityPolicy) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	GetByID(ctx context.Context, id uuid.UUID) (SecurityPolicy, error)
	GetDefault(ctx context.Context) (SecurityPolicy, error)
	List(ctx context.Context) ([]SecurityPolicy, error)

	// SetDefault атомарно снимает флаг со старой default-политики и ставит
	// на указанную. Если id уже default — no-op. Реализуется в одной транзакции.
	SetDefault(ctx context.Context, id uuid.UUID) error
}
