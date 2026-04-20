package dto

import (
	"time"

	"attendance/internal/domain/attendance"
)

// MyAttendanceItem — одна запись в `/students/me/attendance`.
// Отдаём минимум: без submitted_qr_token, без resolved_by (персональные
// данные другого преподавателя в UI студента не нужны).
type MyAttendanceItem struct {
	ID                string    `json:"id"`
	SessionID         string    `json:"session_id"`
	SubmittedAt       time.Time `json:"submitted_at"`
	PreliminaryStatus string    `json:"preliminary_status"`
	FinalStatus       *string   `json:"final_status,omitempty"`
	EffectiveStatus   string    `json:"effective_status"`
}

func MyAttendanceFromDomain(r attendance.Record) MyAttendanceItem {
	var final *string
	if r.FinalStatus != nil {
		v := string(*r.FinalStatus)
		final = &v
	}
	return MyAttendanceItem{
		ID:                r.ID.String(),
		SessionID:         r.SessionID.String(),
		SubmittedAt:       r.SubmittedAt,
		PreliminaryStatus: string(r.PreliminaryStatus),
		FinalStatus:       final,
		EffectiveStatus:   string(r.EffectiveStatus()),
	}
}

// MyAttendanceListResponse — GET /students/me/attendance.
type MyAttendanceListResponse struct {
	Items []MyAttendanceItem `json:"items"`
	Total int                `json:"total"`
}

// MyStatsResponse — GET /students/me/stats.
type MyStatsResponse struct {
	Total          int     `json:"total"`
	Accepted       int     `json:"accepted"`
	NeedsReview    int     `json:"needs_review"`
	Rejected       int     `json:"rejected"`
	AttendanceRate float64 `json:"attendance_rate"` // accepted / total, 0..1
}
