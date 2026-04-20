// Package session — use case'ы управления учебными занятиями.
// Lifecycle: draft → active → closed. Draft можно править/удалить,
// active — только закрывать.
package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain"
	"attendance/internal/domain/audit"
	"attendance/internal/domain/catalog"
	"attendance/internal/domain/policy"
	domainsession "attendance/internal/domain/session"
	"attendance/internal/domain/user"
	"attendance/internal/platform/authctx"
	"attendance/internal/platform/requestmeta"
)

type Deps struct {
	Sessions domainsession.Repository
	Streams  catalog.StreamRepository
	Policies policy.Repository
	Tx       domain.TxRunner
	Audit    *appaudit.Service
	Clock    domain.Clock
	// Rotator — nullable. Если задан, SessionService.Start запускает ротацию
	// QR, Close — останавливает. Без него сессии живут как draft/active/closed
	// без WS-трансляции (полезно для юнит-тестов).
	Rotator domain.RotatorController
}

type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// =========================================================================
// Create (draft)
// =========================================================================

type CreateInput struct {
	CourseID         uuid.UUID
	ClassroomID      *uuid.UUID
	SecurityPolicyID *uuid.UUID // nil → default
	StartsAt         time.Time
	EndsAt           time.Time
	GroupIDs         []uuid.UUID
	QRTTLSeconds     *int // nil → берём из политики
}

// CreateDraft конструирует draft-сессию. Принципал (teacher/admin) становится
// её teacher_id. Валидирует инварианты: время, непустой список групп, каждая
// группа принадлежит какому-либо потоку курса.
func (s *Service) CreateDraft(ctx context.Context, in CreateInput) (domainsession.Session, error) {
	principal, err := authctx.Require(ctx)
	if err != nil {
		return domainsession.Session{}, err
	}

	if !in.EndsAt.After(in.StartsAt) {
		return domainsession.Session{}, domainsession.ErrInvalidTimeRange
	}
	if len(in.GroupIDs) == 0 {
		return domainsession.Session{}, domainsession.ErrGroupsEmpty
	}

	// Политика: если не задана — берём default.
	var pol policy.SecurityPolicy
	if in.SecurityPolicyID != nil {
		pol, err = s.Policies.GetByID(ctx, *in.SecurityPolicyID)
	} else {
		pol, err = s.Policies.GetDefault(ctx)
	}
	if err != nil {
		return domainsession.Session{}, err
	}

	// QR TTL: если клиент не задал — наследуем из политики.
	ttl := pol.Mechanisms.QRTTL.TTLSeconds
	if in.QRTTLSeconds != nil {
		ttl = *in.QRTTLSeconds
	}
	if ttl < domainsession.MinQRTTLSeconds || ttl > domainsession.MaxQRTTLSeconds {
		return domainsession.Session{}, domainsession.ErrInvalidQRTTL
	}

	if err := s.validateGroupsBelongToCourse(ctx, in.CourseID, in.GroupIDs); err != nil {
		return domainsession.Session{}, err
	}

	// Для draft qr_secret генерируем сразу (колонка NOT NULL, CHECK на длину 32).
	// Публично секрет не используется до Start → status=active, но лежит в
	// хранилище. На этапе Start мы его перегенерируем, чтобы никто не мог
	// подсмотреть через SELECT в draft-состоянии.
	secret, err := randomBytes(domainsession.QRSecretLen)
	if err != nil {
		return domainsession.Session{}, err
	}

	sess := domainsession.Session{
		ID:               uuid.New(),
		TeacherID:        principal.UserID,
		CourseID:         in.CourseID,
		ClassroomID:      in.ClassroomID,
		SecurityPolicyID: pol.ID,
		StartsAt:         in.StartsAt,
		EndsAt:           in.EndsAt,
		Status:           domainsession.StatusDraft,
		QRSecret:         secret,
		QRTTLSeconds:     ttl,
		QRCounter:        0,
		GroupIDs:         in.GroupIDs,
		CreatedAt:        s.Clock.Now(ctx),
	}

	err = s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Sessions.Create(txCtx, sess); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action:     audit.ActionSessionCreated,
			EntityType: "session",
			EntityID:   sess.ID.String(),
			Payload: map[string]any{
				"session_id":   sess.ID.String(),
				"course_id":    sess.CourseID.String(),
				"classroom_id": uuidPtrString(sess.ClassroomID),
				"policy_id":    sess.SecurityPolicyID.String(),
				"group_ids":    uuidStrings(sess.GroupIDs),
				"starts_at":    sess.StartsAt,
				"ends_at":      sess.EndsAt,
				"qr_ttl":       sess.QRTTLSeconds,
			},
		})
	})
	if err != nil {
		return domainsession.Session{}, err
	}
	return sess, nil
}

// =========================================================================
// Update (draft only)
// =========================================================================

type UpdateInput struct {
	ClassroomID      *OptUUID
	SecurityPolicyID *uuid.UUID
	StartsAt         *time.Time
	EndsAt           *time.Time
	GroupIDs         *[]uuid.UUID
	QRTTLSeconds     *int
}

// OptUUID позволяет отличить «поле не пришло в PATCH» от «пришло с null-значением».
// Set=true & Value=nil → снять привязку (classroom → онлайн-формат).
type OptUUID struct {
	Set   bool
	Value *uuid.UUID
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (domainsession.Session, error) {
	var out domainsession.Session
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Sessions.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := s.requireTeacherOrAdmin(txCtx, cur); err != nil {
			return err
		}
		if cur.Status != domainsession.StatusDraft {
			return domainsession.ErrInvalidStatusTransition
		}

		if in.ClassroomID != nil && in.ClassroomID.Set {
			cur.ClassroomID = in.ClassroomID.Value
		}
		if in.SecurityPolicyID != nil {
			cur.SecurityPolicyID = *in.SecurityPolicyID
		}
		if in.StartsAt != nil {
			cur.StartsAt = *in.StartsAt
		}
		if in.EndsAt != nil {
			cur.EndsAt = *in.EndsAt
		}
		if in.GroupIDs != nil {
			cur.GroupIDs = *in.GroupIDs
		}
		if in.QRTTLSeconds != nil {
			cur.QRTTLSeconds = *in.QRTTLSeconds
		}

		if !cur.EndsAt.After(cur.StartsAt) {
			return domainsession.ErrInvalidTimeRange
		}
		if cur.QRTTLSeconds < domainsession.MinQRTTLSeconds || cur.QRTTLSeconds > domainsession.MaxQRTTLSeconds {
			return domainsession.ErrInvalidQRTTL
		}
		if len(cur.GroupIDs) == 0 {
			return domainsession.ErrGroupsEmpty
		}
		if err := s.validateGroupsBelongToCourse(txCtx, cur.CourseID, cur.GroupIDs); err != nil {
			return err
		}

		if err := s.Sessions.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionSessionUpdated, EntityType: "session", EntityID: id.String(),
			Payload: map[string]any{
				"session_id":   id.String(),
				"classroom_id": uuidPtrString(cur.ClassroomID),
				"policy_id":    cur.SecurityPolicyID.String(),
				"group_ids":    uuidStrings(cur.GroupIDs),
				"starts_at":    cur.StartsAt,
				"ends_at":      cur.EndsAt,
				"qr_ttl":       cur.QRTTLSeconds,
			},
		})
	})
	return out, err
}

// =========================================================================
// Start (draft → active)
// =========================================================================

func (s *Service) Start(ctx context.Context, id uuid.UUID) (domainsession.Session, error) {
	var out domainsession.Session
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Sessions.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := s.requireTeacherOrAdmin(txCtx, cur); err != nil {
			return err
		}
		if cur.Status != domainsession.StatusDraft {
			return domainsession.ErrInvalidStatusTransition
		}

		// Перегенерируем qr_secret: draft-секрет мог быть "засвечен" в логах /
		// аудите через payload — для активной сессии это недопустимо.
		newSecret, err := randomBytes(domainsession.QRSecretLen)
		if err != nil {
			return err
		}
		cur.QRSecret = newSecret
		cur.QRCounter = 0
		cur.Status = domainsession.StatusActive

		if err := s.Sessions.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionSessionStarted, EntityType: "session", EntityID: id.String(),
			// qr_secret в payload НЕ кладём — это чувствительный ключ.
			Payload: map[string]any{
				"session_id": id.String(),
				"qr_ttl":     cur.QRTTLSeconds,
				"starts_at":  cur.StartsAt,
				"ends_at":    cur.EndsAt,
			},
		})
	})
	if err != nil {
		return out, err
	}
	// После успешного commit — поднимаем ротатор. Если Start ротатора упадёт,
	// логируем и продолжаем: сессия активна, bootstrap при рестарте поднимет.
	if s.Rotator != nil {
		if rerr := s.Rotator.Start(out.ID, out.QRSecret, out.QRTTLSeconds); rerr != nil {
			// Логгера в сервисе нет — используем audit-append для видимости.
			// Но чтобы не замусоривать, просто молча игнорируем: bootstrap починит.
			_ = rerr
		}
	}
	return out, nil
}

// =========================================================================
// Close (active → closed)
// =========================================================================

func (s *Service) Close(ctx context.Context, id uuid.UUID) (domainsession.Session, error) {
	var out domainsession.Session
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Sessions.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := s.requireTeacherOrAdmin(txCtx, cur); err != nil {
			return err
		}
		if cur.Status != domainsession.StatusActive {
			return domainsession.ErrInvalidStatusTransition
		}
		cur.Status = domainsession.StatusClosed
		if err := s.Sessions.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionSessionClosed, EntityType: "session", EntityID: id.String(),
			Payload: map[string]any{"session_id": id.String()},
		})
	})
	if err != nil {
		return out, err
	}
	// После успешного commit — останавливаем ротатор и отключаем WS-клиентов.
	if s.Rotator != nil {
		_ = s.Rotator.Stop(out.ID)
	}
	return out, nil
}

// =========================================================================
// Delete (draft only)
// =========================================================================

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Sessions.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := s.requireTeacherOrAdmin(txCtx, cur); err != nil {
			return err
		}
		if cur.Status != domainsession.StatusDraft {
			return fmt.Errorf("%w: draft only", domainsession.ErrInvalidStatusTransition)
		}
		if err := s.Sessions.Delete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionSessionDeleted, EntityType: "session", EntityID: id.String(),
			Payload: map[string]any{"session_id": id.String()},
		})
	})
}

// =========================================================================
// Read
// =========================================================================

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (domainsession.Session, error) {
	return s.Sessions.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f domainsession.ListFilter) ([]domainsession.Session, int, error) {
	return s.Sessions.List(ctx, f)
}

// =========================================================================
// Helpers
// =========================================================================

// validateGroupsBelongToCourse — ключевой инвариант:
// каждая group в session_groups должна существовать среди
// stream_groups тех streams, у которых course_id = cur.CourseID.
func (s *Service) validateGroupsBelongToCourse(ctx context.Context, courseID uuid.UUID, groups []uuid.UUID) error {
	allowed, err := s.Streams.GroupsForCourse(ctx, courseID)
	if err != nil {
		return err
	}
	allowedSet := make(map[uuid.UUID]struct{}, len(allowed))
	for _, g := range allowed {
		allowedSet[g] = struct{}{}
	}
	for _, g := range groups {
		if _, ok := allowedSet[g]; !ok {
			return domainsession.ErrGroupsNotInCourse
		}
	}
	return nil
}

func (s *Service) requireTeacherOrAdmin(ctx context.Context, sess domainsession.Session) error {
	p, ok := authctx.From(ctx)
	if !ok {
		// Нет principal'а — middleware должен был отсечь раньше; но если вдруг
		// нет, это именно unauthorized, а не forbidden.
		return domainsession.ErrForbidden
	}
	if p.Role == user.RoleAdmin {
		return nil
	}
	if p.Role == user.RoleTeacher && p.UserID == sess.TeacherID {
		return nil
	}
	return domainsession.ErrForbidden
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

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("rand: %w", err)
	}
	return b, nil
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
