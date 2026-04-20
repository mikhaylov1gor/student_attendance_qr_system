package dto

import (
	"time"

	"attendance/internal/domain/policy"
)

// PolicyResponse — ответ API для одной политики.
// Mechanisms отдаются как domain.MechanismsConfig — структура уже JSON-ready
// (имеет json-теги), дублировать DTO-копию нет смысла.
type PolicyResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Mechanisms policy.MechanismsConfig `json:"mechanisms"`
	IsDefault  bool                    `json:"is_default"`
	CreatedAt  time.Time               `json:"created_at"`
}

// PolicyFromDomain — маппер в API-ответ.
func PolicyFromDomain(p policy.SecurityPolicy) PolicyResponse {
	return PolicyResponse{
		ID:         p.ID.String(),
		Name:       p.Name,
		Mechanisms: p.Mechanisms,
		IsDefault:  p.IsDefault,
		CreatedAt:  p.CreatedAt,
	}
}

// CreatePolicyRequest — тело POST /policies.
type CreatePolicyRequest struct {
	Name       string                  `json:"name"       validate:"required,min=1,max=64"`
	Mechanisms policy.MechanismsConfig `json:"mechanisms" validate:"required"`
	IsDefault  bool                    `json:"is_default"`
}

// UpdatePolicyRequest — тело PATCH /policies/:id. Все поля опциональны.
type UpdatePolicyRequest struct {
	Name       *string                  `json:"name,omitempty"       validate:"omitempty,min=1,max=64"`
	Mechanisms *policy.MechanismsConfig `json:"mechanisms,omitempty"`
}

// PolicyListResponse — тело GET /policies.
type PolicyListResponse struct {
	Items []PolicyResponse `json:"items"`
	Total int              `json:"total"`
}
