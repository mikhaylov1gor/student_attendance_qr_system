package user

import (
	"context"

	"github.com/google/uuid"
)

// ListFilter — параметры выборки списка пользователей.
// Поиск по ФИО (Query) реализуется в application-слое после расшифровки
// (см. план, этап 10): на уровне БД fts по ciphertext невозможен.
type ListFilter struct {
	Role    *Role
	GroupID *uuid.UUID
	Limit   int
	Offset  int
}

// Repository — порт репозитория пользователей.
// Реализация — infrastructure/db/repo/user_repo.go (этап 4).
type Repository interface {
	Create(ctx context.Context, u User) error
	Update(ctx context.Context, u User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)

	List(ctx context.Context, f ListFilter) ([]User, int, error)
}
