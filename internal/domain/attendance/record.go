// Package attendance описывает отметку посещаемости и результаты прогона
// механизмов защиты.
package attendance

import (
	"time"

	"github.com/google/uuid"
)

// Status — единый enum для preliminary_status и final_status.
// На final_status накладывается дополнительное ограничение (см. IsValidFinal):
// needs_review не допускается — преподаватель обязан принять решение.
type Status string

const (
	StatusAccepted    Status = "accepted"
	StatusNeedsReview Status = "needs_review"
	StatusRejected    Status = "rejected"
)

// Valid возвращает true для любого из трёх доменных значений.
func (s Status) Valid() bool {
	switch s {
	case StatusAccepted, StatusNeedsReview, StatusRejected:
		return true
	}
	return false
}

// IsValidFinal возвращает true, если значение допустимо как final_status,
// т.е. не равно needs_review. Консистентно с CHECK attendance_final_status_check
// в миграции 0001.
func (s Status) IsValidFinal() bool {
	return s == StatusAccepted || s == StatusRejected
}

// Record — доменная сущность отметки посещаемости.
//
// Инвариант (см. CHECK attendance_resolved_consistency_check в миграции 0001):
// FinalStatus, ResolvedBy, ResolvedAt заполняются/обнуляются тройкой —
// либо всё nil, либо всё заполнено.
type Record struct {
	ID                uuid.UUID
	SessionID         uuid.UUID
	StudentID         uuid.UUID
	SubmittedAt       time.Time
	SubmittedQRToken  string
	PreliminaryStatus Status
	FinalStatus       *Status
	ResolvedBy        *uuid.UUID
	ResolvedAt        *time.Time
	Notes             string
}

// EffectiveStatus возвращает финальный статус, если он выставлен преподавателем,
// иначе — предварительный (автоматический). Используется в отчётах и UI.
func (r Record) EffectiveStatus() Status {
	if r.FinalStatus != nil {
		return *r.FinalStatus
	}
	return r.PreliminaryStatus
}
