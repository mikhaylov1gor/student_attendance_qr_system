// Package catalog — use case'ы CRUD справочников: course, group, stream, classroom.
// Все мутации оборачиваются в Tx.Run: основное изменение + audit.Append атомарно.
package catalog

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain"
	"attendance/internal/domain/audit"
	"attendance/internal/domain/catalog"
	"attendance/internal/platform/authctx"
	"attendance/internal/platform/requestmeta"
)

// Deps — все зависимости сервиса: репозитории + Tx + audit + clock.
type Deps struct {
	Courses    catalog.CourseRepository
	Groups     catalog.GroupRepository
	Streams    catalog.StreamRepository
	Classrooms catalog.ClassroomRepository
	Tx         domain.TxRunner
	Audit      *appaudit.Service
	Clock      domain.Clock
}

// Service объединяет CRUD для всех catalog-сущностей.
type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// =========================================================================
// Course
// =========================================================================

type CreateCourseInput struct {
	Name string
	Code string
}

func (s *Service) CreateCourse(ctx context.Context, in CreateCourseInput) (catalog.Course, error) {
	c := catalog.Course{
		ID:        uuid.New(),
		Name:      in.Name,
		Code:      in.Code,
		CreatedAt: s.Clock.Now(ctx),
	}
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Courses.Create(txCtx, c); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogCourseCreated, EntityType: "course", EntityID: c.ID.String(),
			Payload: map[string]any{"id": c.ID.String(), "name": c.Name, "code": c.Code},
		})
	})
	if err != nil {
		return catalog.Course{}, err
	}
	return c, nil
}

type UpdateCourseInput struct {
	Name *string
	Code *string
}

func (s *Service) UpdateCourse(ctx context.Context, id uuid.UUID, in UpdateCourseInput) (catalog.Course, error) {
	var out catalog.Course
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Courses.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if in.Name != nil {
			cur.Name = *in.Name
		}
		if in.Code != nil {
			cur.Code = *in.Code
		}
		if err := s.Courses.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogCourseUpdated, EntityType: "course", EntityID: id.String(),
			Payload: map[string]any{"id": id.String(), "name": cur.Name, "code": cur.Code},
		})
	})
	return out, err
}

func (s *Service) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Courses.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogCourseDeleted, EntityType: "course", EntityID: id.String(),
			Payload: map[string]any{"id": id.String()},
		})
	})
}

func (s *Service) GetCourse(ctx context.Context, id uuid.UUID) (catalog.Course, error) {
	return s.Courses.GetByID(ctx, id)
}
func (s *Service) ListCourses(ctx context.Context) ([]catalog.Course, error) {
	return s.Courses.List(ctx)
}

// =========================================================================
// Group
// =========================================================================

type CreateGroupInput struct{ Name string }

func (s *Service) CreateGroup(ctx context.Context, in CreateGroupInput) (catalog.Group, error) {
	g := catalog.Group{
		ID:        uuid.New(),
		Name:      in.Name,
		CreatedAt: s.Clock.Now(ctx),
	}
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Groups.Create(txCtx, g); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogGroupCreated, EntityType: "group", EntityID: g.ID.String(),
			Payload: map[string]any{"id": g.ID.String(), "name": g.Name},
		})
	})
	if err != nil {
		return catalog.Group{}, err
	}
	return g, nil
}

type UpdateGroupInput struct{ Name *string }

func (s *Service) UpdateGroup(ctx context.Context, id uuid.UUID, in UpdateGroupInput) (catalog.Group, error) {
	var out catalog.Group
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Groups.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if in.Name != nil {
			cur.Name = *in.Name
		}
		if err := s.Groups.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogGroupUpdated, EntityType: "group", EntityID: id.String(),
			Payload: map[string]any{"id": id.String(), "name": cur.Name},
		})
	})
	return out, err
}

func (s *Service) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Groups.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogGroupDeleted, EntityType: "group", EntityID: id.String(),
			Payload: map[string]any{"id": id.String()},
		})
	})
}

func (s *Service) GetGroup(ctx context.Context, id uuid.UUID) (catalog.Group, error) {
	return s.Groups.GetByID(ctx, id)
}
func (s *Service) ListGroups(ctx context.Context) ([]catalog.Group, error) {
	return s.Groups.List(ctx)
}

// =========================================================================
// Stream
// =========================================================================

type CreateStreamInput struct {
	CourseID uuid.UUID
	Name     string
	GroupIDs []uuid.UUID
}

func (s *Service) CreateStream(ctx context.Context, in CreateStreamInput) (catalog.Stream, error) {
	st := catalog.Stream{
		ID:        uuid.New(),
		CourseID:  in.CourseID,
		Name:      in.Name,
		GroupIDs:  in.GroupIDs,
		CreatedAt: s.Clock.Now(ctx),
	}
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		// Существование course и каждой group — доверяем FK/NOT NULL БД.
		if err := s.Streams.Create(txCtx, st); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogStreamCreated, EntityType: "stream", EntityID: st.ID.String(),
			Payload: map[string]any{
				"id":        st.ID.String(),
				"course_id": st.CourseID.String(),
				"name":      st.Name,
				"group_ids": uuidStrings(st.GroupIDs),
			},
		})
	})
	if err != nil {
		return catalog.Stream{}, err
	}
	return st, nil
}

type UpdateStreamInput struct {
	Name     *string
	GroupIDs *[]uuid.UUID
}

func (s *Service) UpdateStream(ctx context.Context, id uuid.UUID, in UpdateStreamInput) (catalog.Stream, error) {
	var out catalog.Stream
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Streams.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if in.Name != nil {
			cur.Name = *in.Name
		}
		if in.GroupIDs != nil {
			cur.GroupIDs = *in.GroupIDs
		}
		if err := s.Streams.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogStreamUpdated, EntityType: "stream", EntityID: id.String(),
			Payload: map[string]any{
				"id":        id.String(),
				"name":      cur.Name,
				"group_ids": uuidStrings(cur.GroupIDs),
			},
		})
	})
	return out, err
}

func (s *Service) DeleteStream(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Streams.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogStreamDeleted, EntityType: "stream", EntityID: id.String(),
			Payload: map[string]any{"id": id.String()},
		})
	})
}

func (s *Service) GetStream(ctx context.Context, id uuid.UUID) (catalog.Stream, error) {
	return s.Streams.GetByID(ctx, id)
}
func (s *Service) ListStreamsByCourse(ctx context.Context, courseID uuid.UUID) ([]catalog.Stream, error) {
	return s.Streams.ListByCourse(ctx, courseID)
}

// =========================================================================
// Classroom
// =========================================================================

type CreateClassroomInput struct {
	Building      string
	RoomNumber    string
	Latitude      float64
	Longitude     float64
	RadiusMeters  int
	AllowedBSSIDs []string
}

func (s *Service) CreateClassroom(ctx context.Context, in CreateClassroomInput) (catalog.Classroom, error) {
	c := catalog.Classroom{
		ID:            uuid.New(),
		Building:      in.Building,
		RoomNumber:    in.RoomNumber,
		Latitude:      in.Latitude,
		Longitude:     in.Longitude,
		RadiusMeters:  in.RadiusMeters,
		AllowedBSSIDs: in.AllowedBSSIDs,
		CreatedAt:     s.Clock.Now(ctx),
	}
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Classrooms.Create(txCtx, c); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogClassroomCreated, EntityType: "classroom", EntityID: c.ID.String(),
			Payload: map[string]any{
				"id":             c.ID.String(),
				"building":       c.Building,
				"room_number":    c.RoomNumber,
				"radius_m":       c.RadiusMeters,
				"allowed_bssids": c.AllowedBSSIDs,
			},
		})
	})
	if err != nil {
		return catalog.Classroom{}, err
	}
	return c, nil
}

type UpdateClassroomInput struct {
	Building      *string
	RoomNumber    *string
	Latitude      *float64
	Longitude     *float64
	RadiusMeters  *int
	AllowedBSSIDs *[]string
}

func (s *Service) UpdateClassroom(ctx context.Context, id uuid.UUID, in UpdateClassroomInput) (catalog.Classroom, error) {
	var out catalog.Classroom
	err := s.Tx.Run(ctx, func(txCtx context.Context) error {
		cur, err := s.Classrooms.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if in.Building != nil {
			cur.Building = *in.Building
		}
		if in.RoomNumber != nil {
			cur.RoomNumber = *in.RoomNumber
		}
		if in.Latitude != nil {
			cur.Latitude = *in.Latitude
		}
		if in.Longitude != nil {
			cur.Longitude = *in.Longitude
		}
		if in.RadiusMeters != nil {
			cur.RadiusMeters = *in.RadiusMeters
		}
		if in.AllowedBSSIDs != nil {
			cur.AllowedBSSIDs = *in.AllowedBSSIDs
		}
		if err := s.Classrooms.Update(txCtx, cur); err != nil {
			return err
		}
		out = cur
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogClassroomUpdated, EntityType: "classroom", EntityID: id.String(),
			Payload: map[string]any{
				"id":             id.String(),
				"building":       cur.Building,
				"room_number":    cur.RoomNumber,
				"radius_m":       cur.RadiusMeters,
				"allowed_bssids": cur.AllowedBSSIDs,
			},
		})
	})
	return out, err
}

func (s *Service) DeleteClassroom(ctx context.Context, id uuid.UUID) error {
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Classrooms.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			Action: audit.ActionCatalogClassroomDeleted, EntityType: "classroom", EntityID: id.String(),
			Payload: map[string]any{"id": id.String()},
		})
	})
}

func (s *Service) GetClassroom(ctx context.Context, id uuid.UUID) (catalog.Classroom, error) {
	return s.Classrooms.GetByID(ctx, id)
}
func (s *Service) ListClassrooms(ctx context.Context) ([]catalog.Classroom, error) {
	return s.Classrooms.List(ctx)
}

// =========================================================================
// helpers
// =========================================================================

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

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// guard против impoprt-removal при рефакторинге (fmt может стать неиспользуемым).
var _ = fmt.Sprintf
