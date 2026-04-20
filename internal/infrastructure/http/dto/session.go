package dto

import (
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain/session"
)

type SessionResponse struct {
	ID               string    `json:"id"`
	TeacherID        string    `json:"teacher_id"`
	CourseID         string    `json:"course_id"`
	ClassroomID      *string   `json:"classroom_id,omitempty"`
	SecurityPolicyID string    `json:"security_policy_id"`
	StartsAt         time.Time `json:"starts_at"`
	EndsAt           time.Time `json:"ends_at"`
	Status           string    `json:"status"`
	QRTTLSeconds     int       `json:"qr_ttl_seconds"`
	QRCounter        int       `json:"qr_counter"`
	GroupIDs         []string  `json:"group_ids"`
	CreatedAt        time.Time `json:"created_at"`
}

func SessionFromDomain(s session.Session) SessionResponse {
	gids := make([]string, len(s.GroupIDs))
	for i, g := range s.GroupIDs {
		gids[i] = g.String()
	}
	var cid *string
	if s.ClassroomID != nil {
		v := s.ClassroomID.String()
		cid = &v
	}
	return SessionResponse{
		ID:               s.ID.String(),
		TeacherID:        s.TeacherID.String(),
		CourseID:         s.CourseID.String(),
		ClassroomID:      cid,
		SecurityPolicyID: s.SecurityPolicyID.String(),
		StartsAt:         s.StartsAt,
		EndsAt:           s.EndsAt,
		Status:           string(s.Status),
		QRTTLSeconds:     s.QRTTLSeconds,
		QRCounter:        s.QRCounter,
		GroupIDs:         gids,
		CreatedAt:        s.CreatedAt,
	}
}

type CreateSessionRequest struct {
	CourseID         uuid.UUID   `json:"course_id"          validate:"required"`
	ClassroomID      *uuid.UUID  `json:"classroom_id,omitempty"`
	SecurityPolicyID *uuid.UUID  `json:"security_policy_id,omitempty"`
	StartsAt         time.Time   `json:"starts_at"          validate:"required"`
	EndsAt           time.Time   `json:"ends_at"            validate:"required,gtfield=StartsAt"`
	GroupIDs         []uuid.UUID `json:"group_ids"          validate:"required,min=1,dive,required"`
	QRTTLSeconds     *int        `json:"qr_ttl_seconds,omitempty" validate:"omitempty,min=3,max=120"`
}

// UpdateSessionRequest — все поля опциональны.
// classroom_id — указатель на указатель: `null` → снять привязку; опущен → не менять.
// В Go это выражается через json.RawMessage либо специальный флаг; для простоты
// используем отдельный бул Unset (если ClassroomID=nil и ClassroomIDSet=true,
// значит просили снять).
type UpdateSessionRequest struct {
	ClassroomID      *uuid.UUID   `json:"classroom_id,omitempty"`
	ClearClassroom   bool         `json:"clear_classroom,omitempty"`
	SecurityPolicyID *uuid.UUID   `json:"security_policy_id,omitempty"`
	StartsAt         *time.Time   `json:"starts_at,omitempty"`
	EndsAt           *time.Time   `json:"ends_at,omitempty"`
	GroupIDs         *[]uuid.UUID `json:"group_ids,omitempty" validate:"omitempty,min=1,dive,required"`
	QRTTLSeconds     *int         `json:"qr_ttl_seconds,omitempty" validate:"omitempty,min=3,max=120"`
}

type SessionListResponse struct {
	Items []SessionResponse `json:"items"`
	Total int               `json:"total"`
}
