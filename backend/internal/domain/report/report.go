// Package report содержит доменные типы отчёта по посещаемости.
// Сам отчёт — плоская строка (ReportRow) — результат denormalize-join'а
// attendance_records × users × sessions × courses + check-results.
package report

import (
	"time"

	"github.com/google/uuid"
)

// Filter — параметры выборки. Обязательно ровно один из FilterBy ID полей
// (валидируется на уровне сервиса).
type Filter struct {
	SessionID *uuid.UUID
	GroupID   *uuid.UUID
	CourseID  *uuid.UUID

	// From/To — опциональное ограничение по времени сессии.
	From *time.Time
	To   *time.Time

	// TeacherID — автоматически проставляется для role=teacher, чтобы
	// teacher не мог видеть чужие сессии. Admin передаёт nil.
	TeacherID *uuid.UUID
}

// ReportRow — одна строка отчёта. Student ФИО и email — уже plaintext
// (расшифровано сервисом). Статусы чеков — "" если такого чека не было.
type ReportRow struct {
	AttendanceID uuid.UUID
	SessionID    uuid.UUID

	StudentEmail    string
	StudentFullName string

	CourseName      string
	CourseCode      string
	SessionStartsAt time.Time
	SessionEndsAt   time.Time

	SubmittedAt       time.Time
	PreliminaryStatus string
	FinalStatus       string // "" если не выставлен
	EffectiveStatus   string

	QRTTLStatus string
	GeoStatus   string
	WiFiStatus  string
}
