package policy

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain"
	"attendance/internal/domain/audit"
	"attendance/internal/platform/authctx"
	"attendance/internal/platform/requestmeta"

	domainpolicy "attendance/internal/domain/policy"
)

// Deps — зависимости use case'а.
type Deps struct {
	Repo  domainpolicy.Repository
	Clock domain.Clock
	Tx    domain.TxRunner
	Audit *appaudit.Service
}

// Service — CRUD-сервис политик.
type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// CreateInput — входные данные для Create. ID назначается сервисом.
type CreateInput struct {
	Name       string
	Mechanisms domainpolicy.MechanismsConfig
	IsDefault  bool
	CreatedBy  *uuid.UUID
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domainpolicy.SecurityPolicy, error) {
	if err := ValidateMechanisms(in.Mechanisms); err != nil {
		return domainpolicy.SecurityPolicy{}, err
	}
	p := domainpolicy.SecurityPolicy{
		ID:         uuid.New(),
		Name:       in.Name,
		Mechanisms: in.Mechanisms,
		IsDefault:  false,
		CreatedBy:  in.CreatedBy,
		CreatedAt:  s.Clock.Now(ctx),
	}
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Repo.Create(txCtx, p); err != nil {
			return err
		}
		if in.IsDefault {
			if err := s.Repo.SetDefault(txCtx, p.ID); err != nil {
				return fmt.Errorf("set default: %w", err)
			}
			p.IsDefault = true
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionPolicyCreated,
			EntityType: "security_policy",
			EntityID:   p.ID.String(),
			Payload: map[string]any{
				"policy_id":  p.ID.String(),
				"name":       p.Name,
				"is_default": p.IsDefault,
				"mechanisms": p.Mechanisms,
			},
		})
	})
	if err != nil {
		return domainpolicy.SecurityPolicy{}, err
	}
	return p, nil
}

// UpdateInput — частичное обновление. Nil-поля не трогаем.
type UpdateInput struct {
	Name       *string
	Mechanisms *domainpolicy.MechanismsConfig
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (domainpolicy.SecurityPolicy, error) {
	var updated domainpolicy.SecurityPolicy
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		current, err := s.Repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if in.Name != nil {
			current.Name = *in.Name
		}
		if in.Mechanisms != nil {
			if err := ValidateMechanisms(*in.Mechanisms); err != nil {
				return err
			}
			current.Mechanisms = *in.Mechanisms
		}
		if err := s.Repo.Update(txCtx, current); err != nil {
			return err
		}
		updated = current
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionPolicyUpdated,
			EntityType: "security_policy",
			EntityID:   current.ID.String(),
			Payload: map[string]any{
				"policy_id":  current.ID.String(),
				"name":       current.Name,
				"mechanisms": current.Mechanisms,
			},
		})
	})
	if err != nil {
		return domainpolicy.SecurityPolicy{}, err
	}
	return updated, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (domainpolicy.SecurityPolicy, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *Service) GetDefault(ctx context.Context) (domainpolicy.SecurityPolicy, error) {
	return s.Repo.GetDefault(ctx)
}

func (s *Service) List(ctx context.Context) ([]domainpolicy.SecurityPolicy, error) {
	return s.Repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Repo.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionPolicyDeleted,
			EntityType: "security_policy",
			EntityID:   id.String(),
			Payload: map[string]any{
				"policy_id": id.String(),
			},
		})
	})
}

func (s *Service) SetDefault(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Repo.SetDefault(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionPolicyDefaultSet,
			EntityType: "security_policy",
			EntityID:   id.String(),
			Payload: map[string]any{
				"policy_id": id.String(),
			},
		})
	})
}

// auditAppend — дополняет entry actor'ом и request-meta, вызывает Audit.
func (s *Service) auditAppend(ctx context.Context, e audit.Entry) error {
	if s.Audit == nil {
		return nil
	}
	if p, ok := authctx.From(ctx); ok {
		e.ActorID = &p.UserID
		e.ActorRole = string(p.Role)
	}
	meta := requestmeta.From(ctx)
	e.IPAddress = meta.RemoteIP
	e.UserAgent = meta.UserAgent
	_, err := s.Audit.Append(ctx, e)
	return err
}
