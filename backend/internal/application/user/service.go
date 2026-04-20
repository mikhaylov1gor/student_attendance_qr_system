// Package user — use case'ы управления пользователями. Только admin вызывает
// эти методы через REST. Для self-service (смена своего пароля, редактирование
// ФИО) понадобятся отдельные endpoint'ы; здесь их нет — это работа этапа 13+.
package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain"
	"attendance/internal/domain/audit"
	"attendance/internal/domain/user"
	"attendance/internal/platform/authctx"
	"attendance/internal/platform/requestmeta"
)

// Deps — зависимости сервиса.
type Deps struct {
	Users  user.Repository
	Hasher domain.PasswordHasher
	Clock  domain.Clock
	Tx     domain.TxRunner
	Audit  *appaudit.Service
}

type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// =========================================================================
// Create
// =========================================================================

// CreateInput — вход для Create.
//
// Если Password пуст, сервис сам генерирует temp-пароль и возвращает его
// в CreateOutput.TempPassword (показать админу один раз). Для role=student
// GroupID обязателен; для teacher/admin — запрещён.
type CreateInput struct {
	Email    string
	Password string // пусто → генерируется temp
	FullName user.FullName
	Role     user.Role
	GroupID  *uuid.UUID
}

type CreateOutput struct {
	User         user.User
	TempPassword string // не пусто, если генерировали сами
}

func (s *Service) Create(ctx context.Context, in CreateInput) (CreateOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		return CreateOutput{}, fmt.Errorf("user: email required")
	}
	if !in.Role.Valid() {
		return CreateOutput{}, user.ErrInvalidRole
	}
	if err := validateRoleGroup(in.Role, in.GroupID); err != nil {
		return CreateOutput{}, err
	}

	password := in.Password
	temp := ""
	if password == "" {
		var err error
		temp, err = GenerateTempPassword()
		if err != nil {
			return CreateOutput{}, err
		}
		password = temp
	}

	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return CreateOutput{}, fmt.Errorf("user: hash: %w", err)
	}

	u := user.User{
		ID:             uuid.New(),
		Email:          email,
		PasswordHash:   hash,
		FullName:       in.FullName,
		Role:           in.Role,
		CurrentGroupID: in.GroupID,
		CreatedAt:      s.Clock.Now(ctx),
	}

	err = s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Users.Create(txCtx, u); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionUserCreated,
			EntityType: "user",
			EntityID:   u.ID.String(),
			Payload: map[string]any{
				"id":               u.ID.String(),
				"email":            u.Email,
				"role":             string(u.Role),
				"current_group_id": uuidPtrString(u.CurrentGroupID),
				"temp_password":    temp != "", // сам пароль в audit НЕ кладём
			},
		})
	})
	if err != nil {
		return CreateOutput{}, err
	}
	return CreateOutput{User: u, TempPassword: temp}, nil
}

// =========================================================================
// Update
// =========================================================================

// UpdateInput — частичное обновление. Nil-поле не меняет текущее значение.
// ClearGroup=true (в паре с GroupID=nil) отличает «не трогать» от «снять».
type UpdateInput struct {
	Email      *string
	FullName   *user.FullName
	Role       *user.Role
	GroupID    *uuid.UUID
	ClearGroup bool
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (user.User, error) {
	var out user.User
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Users.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if in.Email != nil {
			cur.Email = strings.ToLower(strings.TrimSpace(*in.Email))
		}
		if in.FullName != nil {
			cur.FullName = *in.FullName
		}
		if in.Role != nil {
			if !in.Role.Valid() {
				return user.ErrInvalidRole
			}
			cur.Role = *in.Role
		}
		// Разбор намерения по GroupID: три варианта —
		//   1) in.GroupID != nil          → установить именно этот uuid;
		//   2) in.ClearGroup == true       → снять привязку (nil);
		//   3) иначе — не трогать.
		switch {
		case in.GroupID != nil:
			cur.CurrentGroupID = in.GroupID
		case in.ClearGroup:
			cur.CurrentGroupID = nil
		}
		// Финальная проверка инварианта role↔group.
		if err := validateRoleGroup(cur.Role, cur.CurrentGroupID); err != nil {
			return err
		}
		if err := s.Users.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionUserUpdated,
			EntityType: "user",
			EntityID:   id.String(),
			Payload: map[string]any{
				"id":               id.String(),
				"email":            cur.Email,
				"role":             string(cur.Role),
				"current_group_id": uuidPtrString(cur.CurrentGroupID),
			},
		})
	})
	return out, err
}

// =========================================================================
// Delete (soft)
// =========================================================================

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Users.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionUserDeleted,
			EntityType: "user",
			EntityID:   id.String(),
			Payload:    map[string]any{"id": id.String()},
		})
	})
}

// =========================================================================
// ResetPassword
// =========================================================================

// ResetPassword генерирует новый temp-пароль, хеширует, пишет, возвращает
// открытое значение (показать админу один раз).
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID) (string, error) {
	var tempPass string
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Users.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		newPass, err := GenerateTempPassword()
		if err != nil {
			return err
		}
		hash, err := s.Hasher.Hash(newPass)
		if err != nil {
			return fmt.Errorf("user: hash: %w", err)
		}
		cur.PasswordHash = hash
		if err := s.Users.Update(txCtx, cur); err != nil {
			return err
		}
		tempPass = newPass
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionUserPasswordReset,
			EntityType: "user",
			EntityID:   id.String(),
			Payload:    map[string]any{"id": id.String()},
		})
	})
	if err != nil {
		return "", err
	}
	return tempPass, nil
}

// =========================================================================
// Read
// =========================================================================

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	return s.Users.GetByID(ctx, id)
}

// ListFilter — фильтр + поиск по ФИО.
// Query (фамилия/имя/отчество) фильтруется в памяти после расшифровки —
// ФИО хранится в шифрованных колонках, SQL ILIKE невозможен.
type ListFilter struct {
	Role    *user.Role
	GroupID *uuid.UUID
	Query   string
	Limit   int
	Offset  int
}

// ListWithSearch возвращает пользователей по фильтру + подстроковому совпадению
// в ФИО (case-insensitive). Поскольку ФИО зашифровано, поиск идёт in-memory
// после расшифровки — для dev-объёмов (сотни пользователей) допустимо.
func (s *Service) ListWithSearch(ctx context.Context, f ListFilter) ([]user.User, int, error) {
	// В репо идут только Role/GroupID/Limit/Offset — поиск по Query делаем здесь.
	// Для корректной пагинации при поиске:
	//   - если Query пусто — используем limit/offset БД;
	//   - если Query есть — грузим «с запасом», фильтруем, потом обрезаем.
	if f.Query == "" {
		return s.Users.List(ctx, user.ListFilter{
			Role: f.Role, GroupID: f.GroupID,
			Limit: f.Limit, Offset: f.Offset,
		})
	}

	// Эвристика: грузим первые 500 записей по фильтру; если найденных меньше —
	// пагинация корректна. Для prod надо бы полнотекстовый индекс (future work).
	const searchCap = 500
	all, total, err := s.Users.List(ctx, user.ListFilter{
		Role: f.Role, GroupID: f.GroupID,
		Limit: searchCap, Offset: 0,
	})
	if err != nil {
		return nil, 0, err
	}
	needle := strings.ToLower(strings.TrimSpace(f.Query))
	matched := make([]user.User, 0, len(all))
	for _, u := range all {
		hay := strings.ToLower(u.FullName.String())
		if strings.Contains(hay, needle) {
			matched = append(matched, u)
		}
	}
	// Manual pagination среди совпадений.
	start := f.Offset
	if start > len(matched) {
		start = len(matched)
	}
	end := len(matched)
	if f.Limit > 0 && start+f.Limit < end {
		end = start + f.Limit
	}
	_ = total // total в режиме поиска не имеет полного смысла — отдаём len(matched) как approximate
	return matched[start:end], len(matched), nil
}

// =========================================================================
// helpers
// =========================================================================

// validateRoleGroup проверяет инвариант role↔current_group_id.
func validateRoleGroup(r user.Role, gid *uuid.UUID) error {
	if r == user.RoleStudent && gid == nil {
		return user.ErrRoleGroupMismatch
	}
	if r != user.RoleStudent && gid != nil {
		return user.ErrRoleGroupMismatch
	}
	return nil
}

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

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

// Проверочная ссылка, чтобы errors-импорт не остался висячим, если все
// текущие места перейдут на sentinel-сравнения.
var _ = errors.Is
