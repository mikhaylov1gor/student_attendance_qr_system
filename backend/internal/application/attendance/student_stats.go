package attendance

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"attendance/internal/domain/attendance"
)

// StudentStats — агрегат посещаемости одного студента по всем сессиям.
// Используется эндпоинтом `/students/me/stats`.
//
// `Effective` учитывает final_status если он выставлен (ручное решение
// преподавателя), иначе — preliminary_status. Это же правило считает
// Record.EffectiveStatus() на уровне домена.
type StudentStats struct {
	Total          int     `json:"total"`
	Accepted       int     `json:"accepted"`
	NeedsReview    int     `json:"needs_review"`
	Rejected       int     `json:"rejected"`
	AttendanceRate float64 `json:"attendance_rate"` // accepted / total
}

// ListMyAttendance возвращает отметки данного студента постранично.
// Используется эндпоинтом `/students/me/attendance`.
func (s *Service) ListMyAttendance(
	ctx context.Context,
	studentID uuid.UUID,
	limit, offset int,
) ([]attendance.Record, int, error) {
	f := attendance.ListFilter{
		StudentID: &studentID,
		Limit:     limit,
		Offset:    offset,
	}
	records, total, err := s.Attendance.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("list my attendance: %w", err)
	}
	return records, total, nil
}

// GetMyStats агрегирует все отметки студента в StudentStats.
// Для dev-объёмов (десятки-сотни отметок на студента) считаем in-memory.
func (s *Service) GetMyStats(ctx context.Context, studentID uuid.UUID) (StudentStats, error) {
	// Limit=0 → вернуть всё (repo просто не добавит LIMIT).
	records, _, err := s.Attendance.List(ctx, attendance.ListFilter{StudentID: &studentID})
	if err != nil {
		return StudentStats{}, fmt.Errorf("get my stats: %w", err)
	}
	var st StudentStats
	st.Total = len(records)
	for _, r := range records {
		switch r.EffectiveStatus() {
		case attendance.StatusAccepted:
			st.Accepted++
		case attendance.StatusNeedsReview:
			st.NeedsReview++
		case attendance.StatusRejected:
			st.Rejected++
		}
	}
	if st.Total > 0 {
		st.AttendanceRate = float64(st.Accepted) / float64(st.Total)
	}
	return st, nil
}
