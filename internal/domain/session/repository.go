package session

import (
	"context"

	"github.com/google/uuid"
)

// ListFilter — параметры выборки списка сессий.
type ListFilter struct {
	TeacherID *uuid.UUID
	CourseID  *uuid.UUID
	Status    *Status
	FromTime  *string // ISO-8601, парсится в layer выше; держим строкой, чтобы домен не тащил парсеры
	ToTime    *string
	Limit     int
	Offset    int
}

// Repository — порт репозитория сессий. GroupIDs загружаются/сохраняются
// вместе с самой сессией; отдельных методов не выделяем.
type Repository interface {
	Create(ctx context.Context, s Session) error
	Update(ctx context.Context, s Session) error
	Delete(ctx context.Context, id uuid.UUID) error

	GetByID(ctx context.Context, id uuid.UUID) (Session, error)
	List(ctx context.Context, f ListFilter) ([]Session, int, error)

	// ActiveForBootstrap возвращает все сессии со статусом active —
	// используется при старте приложения для восстановления QR-ротатора.
	ActiveForBootstrap(ctx context.Context) ([]Session, error)

	// IncrementQRCounter атомарно увеличивает qr_counter на 1 и возвращает
	// новое значение. Используется ротатором (этап 9).
	IncrementQRCounter(ctx context.Context, id uuid.UUID) (int, error)
}
