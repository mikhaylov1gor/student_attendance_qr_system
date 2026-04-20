package dto

import (
	"time"

	"attendance/internal/domain/attendance"
)

// SubmitAttendanceRequest — тело POST /api/v1/attendance.
// Принципал студента берётся из JWT, сюда не передаётся.
type SubmitAttendanceRequest struct {
	QRToken    string     `json:"qr_token"              validate:"required,min=16,max=256"`
	GeoLat     *float64   `json:"geo_lat,omitempty"     validate:"omitempty,min=-90,max=90"`
	GeoLng     *float64   `json:"geo_lng,omitempty"     validate:"omitempty,min=-180,max=180"`
	BSSID      *string    `json:"bssid,omitempty"       validate:"omitempty,max=64"`
	ClientTime *time.Time `json:"client_time,omitempty"`
}

// ResolveAttendanceRequest — тело PATCH /api/v1/attendance/:id.
// Teacher/admin-only: вручную переводит запись в accepted | rejected.
type ResolveAttendanceRequest struct {
	FinalStatus string `json:"final_status" validate:"required,oneof=accepted rejected"`
	Notes       string `json:"notes,omitempty" validate:"max=500"`
}

// CheckResultResponse — один механизм в ответе.
type CheckResultResponse struct {
	Mechanism string         `json:"mechanism"`
	Status    string         `json:"status"`
	Details   map[string]any `json:"details,omitempty"`
	CheckedAt time.Time      `json:"checked_at"`
}

// AttendanceResponse — успешный /attendance.
// Поля resolved_* заполняются только после teacher-override (PATCH).
type AttendanceResponse struct {
	ID                string                `json:"id"`
	SessionID         string                `json:"session_id"`
	StudentID         string                `json:"student_id"`
	SubmittedAt       time.Time             `json:"submitted_at"`
	PreliminaryStatus string                `json:"preliminary_status"`
	FinalStatus       *string               `json:"final_status,omitempty"`
	ResolvedBy        *string               `json:"resolved_by,omitempty"`
	ResolvedAt        *time.Time            `json:"resolved_at,omitempty"`
	Notes             string                `json:"notes,omitempty"`
	EffectiveStatus   string                `json:"effective_status"`
	Checks            []CheckResultResponse `json:"checks"`
}

func AttendanceFromDomain(r attendance.Record, cs []attendance.CheckResult) AttendanceResponse {
	checks := make([]CheckResultResponse, 0, len(cs))
	for _, c := range cs {
		checks = append(checks, CheckResultResponse{
			Mechanism: c.Mechanism,
			Status:    string(c.Status),
			Details:   c.Details,
			CheckedAt: c.CheckedAt,
		})
	}
	out := AttendanceResponse{
		ID:                r.ID.String(),
		SessionID:         r.SessionID.String(),
		StudentID:         r.StudentID.String(),
		SubmittedAt:       r.SubmittedAt,
		PreliminaryStatus: string(r.PreliminaryStatus),
		EffectiveStatus:   string(r.EffectiveStatus()),
		Notes:             r.Notes,
		Checks:            checks,
	}
	if r.FinalStatus != nil {
		s := string(*r.FinalStatus)
		out.FinalStatus = &s
	}
	if r.ResolvedBy != nil {
		s := r.ResolvedBy.String()
		out.ResolvedBy = &s
	}
	if r.ResolvedAt != nil {
		t := *r.ResolvedAt
		out.ResolvedAt = &t
	}
	return out
}
