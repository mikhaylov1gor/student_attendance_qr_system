// Package session описывает учебное занятие (сессию) и всё, что к нему относится
// на доменном уровне: статус жизненного цикла, параметры QR-ротации, набор групп.
package session

import (
	"time"

	"github.com/google/uuid"
)

// Status — состояние сессии.
//
//	draft  — создана преподавателем, ещё не запущена, QR не генерируется;
//	active — идёт, QR ротируется, приём отметок открыт;
//	closed — завершена, приём отметок закрыт.
type Status string

const (
	StatusDraft  Status = "draft"
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusActive, StatusClosed:
		return true
	}
	return false
}

// QRSecretLen — размер секрета HMAC в байтах. Соответствует CHECK в миграции 0001.
const QRSecretLen = 32

// MinQRTTLSeconds / MaxQRTTLSeconds — допустимый диапазон TTL одного QR
// (см. CHECK в миграции 0001).
const (
	MinQRTTLSeconds = 3
	MaxQRTTLSeconds = 120
)

// Session — доменная сущность учебного занятия.
type Session struct {
	ID               uuid.UUID
	TeacherID        uuid.UUID
	CourseID         uuid.UUID
	ClassroomID      *uuid.UUID // nullable — для онлайн-занятий
	SecurityPolicyID uuid.UUID
	StartsAt         time.Time
	EndsAt           time.Time
	Status           Status
	QRSecret         []byte // 32 байта; заполняется при переходе draft → active
	QRTTLSeconds     int
	QRCounter        int
	GroupIDs         []uuid.UUID
	CreatedAt        time.Time
}

// IsAcceptingAttendance — можно ли в данный момент принять отметку студента.
// Проверка границ времени нужна здесь, а не в БД: между Start и реальной
// отметкой может пройти переход в closed.
func (s Session) IsAcceptingAttendance(now time.Time) bool {
	if s.Status != StatusActive {
		return false
	}
	if now.Before(s.StartsAt) || now.After(s.EndsAt) {
		return false
	}
	return true
}
