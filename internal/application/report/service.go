// Package report — use case генерации отчётов посещаемости.
// Synchronous: блокирующий вызов → handler стримит ответ в HTTP.
package report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain"
	domainattendance "attendance/internal/domain/attendance"
	"attendance/internal/domain/report"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/db/models"
)

// ErrNoFilter — ни один из session/group/course не задан.
var ErrNoFilter = errors.New("report: one of session_id/group_id/course_id is required")

// ErrAmbiguousFilter — передан больше одного из основных фильтров.
var ErrAmbiguousFilter = errors.New("report: only one of session_id/group_id/course_id allowed")

// Deps — зависимости сервиса.
type Deps struct {
	Repo      report.Repository
	Encryptor domain.FieldEncryptor // расшифровка ФИО
}

type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// GenerateAttendance выполняет запрос и возвращает подготовленные строки
// отчёта. ФИО уже plaintext.
//
// Если role=teacher, вызывающий (handler) обязан передать TeacherID в filter.
// Admin передаёт TeacherID=nil.
func (s *Service) GenerateAttendance(ctx context.Context, f report.Filter) ([]report.ReportRow, error) {
	if err := validateFilter(f); err != nil {
		return nil, err
	}
	raw, err := s.Repo.Query(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]report.ReportRow, 0, len(raw))
	for _, r := range raw {
		row, err := s.mapRow(r)
		if err != nil {
			return nil, fmt.Errorf("report: map row %s: %w", r.AttendanceID, err)
		}
		out = append(out, row)
	}
	return out, nil
}

// validateFilter — ровно один основной фильтр.
func validateFilter(f report.Filter) error {
	cnt := 0
	if f.SessionID != nil {
		cnt++
	}
	if f.GroupID != nil {
		cnt++
	}
	if f.CourseID != nil {
		cnt++
	}
	switch cnt {
	case 0:
		return ErrNoFilter
	case 1:
		return nil
	default:
		return ErrAmbiguousFilter
	}
}

// mapRow превращает RawRow (шифрованное ФИО + ISO-строки) в ReportRow.
// Самая дорогая операция здесь — Decrypt для каждой строки. Для типичных
// отчётов (сотни-тысячи строк) — терпимо.
func (s *Service) mapRow(r report.RawRow) (report.ReportRow, error) {
	aid, err := uuid.Parse(r.AttendanceID)
	if err != nil {
		return report.ReportRow{}, fmt.Errorf("parse attendance_id: %w", err)
	}
	sid, err := uuid.Parse(r.SessionID)
	if err != nil {
		return report.ReportRow{}, fmt.Errorf("parse session_id: %w", err)
	}

	fn, err := models.FullNameDecrypt(r.StudentFullNameCiphertext, r.StudentFullNameNonce, s.Encryptor)
	if err != nil {
		return report.ReportRow{}, fmt.Errorf("decrypt fullname: %w", err)
	}

	parseTS := func(s string) (time.Time, error) {
		// Постгрес to_char даёт либо 'YYYY-MM-DDTHH:MI:SSZ', либо с .US — парсим оба.
		if t, err := time.Parse("2006-01-02T15:04:05Z", s); err == nil {
			return t, nil
		}
		return time.Parse("2006-01-02T15:04:05.000000Z", s)
	}
	startsAt, err := parseTS(r.SessionStartsAt)
	if err != nil {
		return report.ReportRow{}, fmt.Errorf("parse starts_at: %w", err)
	}
	endsAt, err := parseTS(r.SessionEndsAt)
	if err != nil {
		return report.ReportRow{}, fmt.Errorf("parse ends_at: %w", err)
	}
	submittedAt, err := parseTS(r.SubmittedAt)
	if err != nil {
		return report.ReportRow{}, fmt.Errorf("parse submitted_at: %w", err)
	}

	final := ""
	if r.FinalStatus != nil {
		final = *r.FinalStatus
	}
	effective := r.PreliminaryStatus
	if final != "" {
		effective = final
	}

	return report.ReportRow{
		AttendanceID:      aid,
		SessionID:         sid,
		StudentEmail:      r.StudentEmail,
		StudentFullName:   fn.String(),
		CourseName:        r.CourseName,
		CourseCode:        r.CourseCode,
		SessionStartsAt:   startsAt,
		SessionEndsAt:     endsAt,
		SubmittedAt:       submittedAt,
		PreliminaryStatus: r.PreliminaryStatus,
		FinalStatus:       final,
		EffectiveStatus:   effective,
		QRTTLStatus:       r.QRTTLStatus,
		GeoStatus:         r.GeoStatus,
		WiFiStatus:        r.WiFiStatus,
	}, nil
}

// DecodeFullName — экспорт для тестов / потенциального переиспользования.
// Не относится напрямую к отчёту, но удобно локально.
var _ = user.FullName{}                 // держим пакет в deps
var _ = domainattendance.StatusAccepted // на случай будущего форматера
