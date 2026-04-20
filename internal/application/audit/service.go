// Package audit — use case'ы tamper-evident журнала.
// Append вызывается внутри транзакции основной мутации (TxRunner.Run).
// Verify пересчитывает цепочку по сохранённым записям.
package audit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"attendance/internal/domain"
	"attendance/internal/domain/audit"
)

// Deps — зависимости audit-сервиса.
type Deps struct {
	Repo  audit.Repository
	Clock domain.Clock
}

type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// Append — записывает событие в журнал.
//
// Порядок:
//  1. SELECT последней записи (advisory_xact_lock внутри Append гарантирует,
//     что никто между нашим чтением и записью не вклинится);
//  2. prev_hash = предыдущая record_hash (или genesis для первой записи);
//  3. record_hash = ComputeRecordHash(prev, entry);
//  4. INSERT.
//
// OccurredAt, если не выставлен, берётся из Clock.
func (s *Service) Append(ctx context.Context, e audit.Entry) (audit.Entry, error) {
	if e.OccurredAt.IsZero() {
		e.OccurredAt = s.Clock.Now(ctx)
	}
	// Критично для цепочки: Go хранит время с наносекундной precision,
	// Postgres timestamptz — с микросекундной. Если хэшировать с nsec, после
	// round-trip через БД verify пересчитает хэш с usec и не сойдётся.
	// Обрезаем до микросекунды в UTC ДО вычисления hash.
	e.OccurredAt = e.OccurredAt.UTC().Truncate(time.Microsecond)

	// Нормализуем: nil payload → пустой объект, чтобы в БД не было NULL'ов
	// (колонка NOT NULL, а домен допускает nil).
	if e.Payload == nil {
		e.Payload = map[string]any{}
	}
	// Нормализуем все time.Time в payload до UTC+microsecond: gorm/json.Marshal
	// сериализует time.Time в локальном TZ, а canonicalize для hash — в UTC.
	// Без нормализации хэш расходится с round-trip'ом через JSONB.
	if norm, ok := normalizePayloadTimes(e.Payload).(map[string]any); ok {
		e.Payload = norm
	}

	last, ok, err := s.Repo.Last(ctx)
	if err != nil {
		return audit.Entry{}, fmt.Errorf("audit append: load last: %w", err)
	}
	var prevHash []byte
	if ok {
		prevHash = last.RecordHash
	} else {
		prevHash = audit.GenesisPrevHash()
	}

	recordHash, err := audit.ComputeRecordHash(prevHash, e)
	if err != nil {
		return audit.Entry{}, fmt.Errorf("audit append: compute hash: %w", err)
	}
	e.PrevHash = prevHash
	e.RecordHash = recordHash

	return s.Repo.Append(ctx, e)
}

// VerifyResult — итог верификации цепочки.
type VerifyResult struct {
	OK            bool
	TotalEntries  int
	FirstBrokenID *int64
	BrokenReason  string
}

// normalizePayloadTimes проходит по payload'у рекурсивно и приводит все
// time.Time к UTC с микросекундной точностью. Без этого шага хэш, посчитанный
// на стороне Append (canonicalize→UTC), не совпадёт с хэшем при Verify: БД
// через JSONB хранит строку, сериализованную json.Marshal'ом в ИСХОДНОМ TZ.
func normalizePayloadTimes(v any) any {
	switch val := v.(type) {
	case time.Time:
		return val.UTC().Truncate(time.Microsecond)
	case map[string]any:
		for k, inner := range val {
			val[k] = normalizePayloadTimes(inner)
		}
		return val
	case []any:
		for i, inner := range val {
			val[i] = normalizePayloadTimes(inner)
		}
		return val
	default:
		return val
	}
}

// Verify пересчитывает hash-chain от начала до конца батчами. Первая
// несовпавшая запись останавливает скан и возвращается в FirstBrokenID.
func (s *Service) Verify(ctx context.Context) (VerifyResult, error) {
	genesis := audit.GenesisPrevHash()
	prev := make([]byte, audit.HashLen)
	copy(prev, genesis)

	res := VerifyResult{OK: true}
	errStop := errors.New("audit: stop scan")

	err := s.Repo.Scan(ctx, 500, func(e audit.Entry) error {
		res.TotalEntries++

		if !bytes.Equal(e.PrevHash, prev) {
			res.OK = false
			id := e.ID
			res.FirstBrokenID = &id
			res.BrokenReason = "prev_hash mismatch"
			return errStop
		}

		expected, err := audit.ComputeRecordHash(prev, e)
		if err != nil {
			return err
		}
		if !bytes.Equal(expected, e.RecordHash) {
			res.OK = false
			id := e.ID
			res.FirstBrokenID = &id
			res.BrokenReason = "record_hash mismatch"
			return errStop
		}

		prev = e.RecordHash
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return VerifyResult{}, err
	}
	return res, nil
}

// List — чтение журнала с фильтрами (для admin-интерфейса).
func (s *Service) List(ctx context.Context, f audit.ListFilter) ([]audit.Entry, int, error) {
	return s.Repo.List(ctx, f)
}
