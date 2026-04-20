package catalog

import (
	"context"

	"github.com/google/uuid"
)

// CourseRepository — CRUD над Course.
type CourseRepository interface {
	Create(ctx context.Context, c Course) error
	Update(ctx context.Context, c Course) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (Course, error)
	GetByCode(ctx context.Context, code string) (Course, error)
	List(ctx context.Context) ([]Course, error)
}

// GroupRepository — CRUD над Group.
type GroupRepository interface {
	Create(ctx context.Context, g Group) error
	Update(ctx context.Context, g Group) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (Group, error)
	List(ctx context.Context) ([]Group, error)
}

// StreamRepository — CRUD над Stream + управление составом (M:N stream_groups).
type StreamRepository interface {
	Create(ctx context.Context, s Stream) error
	Update(ctx context.Context, s Stream) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (Stream, error)

	// ListByCourse возвращает все потоки курса.
	ListByCourse(ctx context.Context, courseID uuid.UUID) ([]Stream, error)

	// GroupsForCourse возвращает множество групп, доступных для сессии по
	// данному курсу (объединение групп всех его потоков). Используется для
	// инварианта «выбранные группы сессии должны принадлежать курсу».
	GroupsForCourse(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error)
}

// ClassroomRepository — CRUD над Classroom.
type ClassroomRepository interface {
	Create(ctx context.Context, c Classroom) error
	Update(ctx context.Context, c Classroom) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (Classroom, error)
	List(ctx context.Context) ([]Classroom, error)
}
