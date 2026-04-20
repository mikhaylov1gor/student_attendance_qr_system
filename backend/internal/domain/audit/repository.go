package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListFilter — параметры выборки журнала.
type ListFilter struct {
	ActorID    *uuid.UUID
	Action     *Action
	EntityType *string
	EntityID   *string
	FromTime   *time.Time
	ToTime     *time.Time
	Limit      int
	Offset     int
}

// Repository — порт журнала аудита.
//
// Append обязан выполнять SELECT последней записи + вычисление + INSERT в
// одной транзакции с основным действием. Как это реализуется (advisory-lock,
// serializable isolation) — детали инфраструктуры.
type Repository interface {
	Append(ctx context.Context, e Entry) (Entry, error)

	Last(ctx context.Context) (Entry, bool, error)

	List(ctx context.Context, f ListFilter) ([]Entry, int, error)

	// Scan пробегает цепочку от начала к концу батчами, вызывая fn для каждой
	// записи. Используется верификатором цепочки (этап 7).
	Scan(ctx context.Context, batchSize int, fn func(Entry) error) error
}
