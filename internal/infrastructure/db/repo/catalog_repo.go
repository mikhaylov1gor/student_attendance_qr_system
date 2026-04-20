package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"attendance/internal/domain/catalog"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

// ============================================================================
// CourseRepo
// ============================================================================

type CourseRepo struct{ db *gorm.DB }

func NewCourseRepo(db *gorm.DB) *CourseRepo { return &CourseRepo{db: db} }

var _ catalog.CourseRepository = (*CourseRepo)(nil)

func (r *CourseRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

func (r *CourseRepo) Create(ctx context.Context, c catalog.Course) error {
	m := models.CourseToModel(c)
	if err := r.dbx(ctx).Create(&m).Error; err != nil {
		if isUniqueViolation(err, "courses_code_key") {
			return catalog.ErrCourseCodeTaken
		}
		return fmt.Errorf("course create: %w", err)
	}
	return nil
}

func (r *CourseRepo) Update(ctx context.Context, c catalog.Course) error {
	tx := r.dbx(ctx).
		Model(&models.CourseModel{}).
		Where("id = ? AND deleted_at IS NULL", c.ID).
		Updates(map[string]any{"name": c.Name, "code": c.Code})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error, "courses_code_key") {
			return catalog.ErrCourseCodeTaken
		}
		return fmt.Errorf("course update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrCourseNotFound
	}
	return nil
}

func (r *CourseRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tx := r.dbx(ctx).
		Model(&models.CourseModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("now()"))
	if tx.Error != nil {
		return fmt.Errorf("course delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrCourseNotFound
	}
	return nil
}

func (r *CourseRepo) GetByID(ctx context.Context, id uuid.UUID) (catalog.Course, error) {
	var m models.CourseModel
	err := r.dbx(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalog.Course{}, catalog.ErrCourseNotFound
		}
		return catalog.Course{}, fmt.Errorf("course get: %w", err)
	}
	return models.CourseFromModel(m), nil
}

func (r *CourseRepo) GetByCode(ctx context.Context, code string) (catalog.Course, error) {
	var m models.CourseModel
	err := r.dbx(ctx).Where("code = ? AND deleted_at IS NULL", code).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalog.Course{}, catalog.ErrCourseNotFound
		}
		return catalog.Course{}, fmt.Errorf("course get by code: %w", err)
	}
	return models.CourseFromModel(m), nil
}

func (r *CourseRepo) List(ctx context.Context) ([]catalog.Course, error) {
	var ms []models.CourseModel
	err := r.dbx(ctx).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("course list: %w", err)
	}
	out := make([]catalog.Course, len(ms))
	for i, m := range ms {
		out[i] = models.CourseFromModel(m)
	}
	return out, nil
}

// ============================================================================
// GroupRepo
// ============================================================================

type GroupRepo struct{ db *gorm.DB }

func NewGroupRepo(db *gorm.DB) *GroupRepo { return &GroupRepo{db: db} }

var _ catalog.GroupRepository = (*GroupRepo)(nil)

func (r *GroupRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

func (r *GroupRepo) Create(ctx context.Context, g catalog.Group) error {
	m := models.GroupToModel(g)
	if err := r.dbx(ctx).Create(&m).Error; err != nil {
		if isUniqueViolation(err, "groups_name_key") {
			return catalog.ErrGroupNameTaken
		}
		return fmt.Errorf("group create: %w", err)
	}
	return nil
}

func (r *GroupRepo) Update(ctx context.Context, g catalog.Group) error {
	tx := r.dbx(ctx).
		Model(&models.GroupModel{}).
		Where("id = ? AND deleted_at IS NULL", g.ID).
		Updates(map[string]any{"name": g.Name})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error, "groups_name_key") {
			return catalog.ErrGroupNameTaken
		}
		return fmt.Errorf("group update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrGroupNotFound
	}
	return nil
}

func (r *GroupRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tx := r.dbx(ctx).
		Model(&models.GroupModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("now()"))
	if tx.Error != nil {
		return fmt.Errorf("group delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrGroupNotFound
	}
	return nil
}

func (r *GroupRepo) GetByID(ctx context.Context, id uuid.UUID) (catalog.Group, error) {
	var m models.GroupModel
	err := r.dbx(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalog.Group{}, catalog.ErrGroupNotFound
		}
		return catalog.Group{}, fmt.Errorf("group get: %w", err)
	}
	return models.GroupFromModel(m), nil
}

func (r *GroupRepo) List(ctx context.Context) ([]catalog.Group, error) {
	var ms []models.GroupModel
	err := r.dbx(ctx).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("group list: %w", err)
	}
	out := make([]catalog.Group, len(ms))
	for i, m := range ms {
		out[i] = models.GroupFromModel(m)
	}
	return out, nil
}

// ============================================================================
// StreamRepo — CRUD + управление составом stream_groups
// ============================================================================

type StreamRepo struct{ db *gorm.DB }

func NewStreamRepo(db *gorm.DB) *StreamRepo { return &StreamRepo{db: db} }

var _ catalog.StreamRepository = (*StreamRepo)(nil)

func (r *StreamRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

// Create: одной транзакцией пишем поток и его состав.
func (r *StreamRepo) Create(ctx context.Context, s catalog.Stream) error {
	m := models.StreamToModel(s)
	return r.dbx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&m).Error; err != nil {
			return fmt.Errorf("stream create: %w", err)
		}
		if len(s.GroupIDs) == 0 {
			return nil
		}
		rows := make([]models.StreamGroupModel, len(s.GroupIDs))
		for i, gid := range s.GroupIDs {
			rows[i] = models.StreamGroupModel{StreamID: m.ID, GroupID: gid}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("stream groups insert: %w", err)
		}
		return nil
	})
}

// Update: обновляем метаданные потока и пересобираем состав.
func (r *StreamRepo) Update(ctx context.Context, s catalog.Stream) error {
	return r.dbx(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.StreamModel{}).
			Where("id = ? AND deleted_at IS NULL", s.ID).
			Updates(map[string]any{"course_id": s.CourseID, "name": s.Name})
		if res.Error != nil {
			return fmt.Errorf("stream update: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return catalog.ErrStreamNotFound
		}
		if err := tx.
			Where("stream_id = ?", s.ID).
			Delete(&models.StreamGroupModel{}).Error; err != nil {
			return fmt.Errorf("stream groups clear: %w", err)
		}
		if len(s.GroupIDs) > 0 {
			rows := make([]models.StreamGroupModel, len(s.GroupIDs))
			for i, gid := range s.GroupIDs {
				rows[i] = models.StreamGroupModel{StreamID: s.ID, GroupID: gid}
			}
			if err := tx.Create(&rows).Error; err != nil {
				return fmt.Errorf("stream groups insert: %w", err)
			}
		}
		return nil
	})
}

func (r *StreamRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tx := r.dbx(ctx).
		Model(&models.StreamModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("now()"))
	if tx.Error != nil {
		return fmt.Errorf("stream delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrStreamNotFound
	}
	return nil
}

func (r *StreamRepo) GetByID(ctx context.Context, id uuid.UUID) (catalog.Stream, error) {
	var m models.StreamModel
	err := r.dbx(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalog.Stream{}, catalog.ErrStreamNotFound
		}
		return catalog.Stream{}, fmt.Errorf("stream get: %w", err)
	}
	groups, err := r.groupsOf(ctx, id)
	if err != nil {
		return catalog.Stream{}, err
	}
	return models.StreamFromModel(m, groups), nil
}

func (r *StreamRepo) ListByCourse(ctx context.Context, courseID uuid.UUID) ([]catalog.Stream, error) {
	var ms []models.StreamModel
	err := r.dbx(ctx).
		Where("course_id = ? AND deleted_at IS NULL", courseID).
		Order("name ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("stream list: %w", err)
	}
	out := make([]catalog.Stream, 0, len(ms))
	for _, m := range ms {
		groups, err := r.groupsOf(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, models.StreamFromModel(m, groups))
	}
	return out, nil
}

func (r *StreamRepo) GroupsForCourse(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.dbx(ctx).
		Table("stream_groups AS sg").
		Joins("JOIN streams AS s ON s.id = sg.stream_id AND s.deleted_at IS NULL").
		Where("s.course_id = ?", courseID).
		Distinct("sg.group_id").
		Pluck("sg.group_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("groups for course: %w", err)
	}
	return ids, nil
}

func (r *StreamRepo) groupsOf(ctx context.Context, streamID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.dbx(ctx).
		Table("stream_groups").
		Where("stream_id = ?", streamID).
		Order("group_id").
		Pluck("group_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("stream groups load: %w", err)
	}
	return ids, nil
}

// ============================================================================
// ClassroomRepo
// ============================================================================

type ClassroomRepo struct{ db *gorm.DB }

func NewClassroomRepo(db *gorm.DB) *ClassroomRepo { return &ClassroomRepo{db: db} }

var _ catalog.ClassroomRepository = (*ClassroomRepo)(nil)

func (r *ClassroomRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

func (r *ClassroomRepo) Create(ctx context.Context, c catalog.Classroom) error {
	m, err := models.ClassroomToModel(c)
	if err != nil {
		return err
	}
	if err := r.dbx(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("classroom create: %w", err)
	}
	return nil
}

func (r *ClassroomRepo) Update(ctx context.Context, c catalog.Classroom) error {
	m, err := models.ClassroomToModel(c)
	if err != nil {
		return err
	}
	tx := r.dbx(ctx).
		Model(&models.ClassroomModel{}).
		Where("id = ? AND deleted_at IS NULL", c.ID).
		Updates(map[string]any{
			"building":       m.Building,
			"room_number":    m.RoomNumber,
			"latitude":       m.Latitude,
			"longitude":      m.Longitude,
			"radius_m":       m.RadiusM,
			"allowed_bssids": m.AllowedBSSIDs,
		})
	if tx.Error != nil {
		return fmt.Errorf("classroom update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrClassroomNotFound
	}
	return nil
}

func (r *ClassroomRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tx := r.dbx(ctx).
		Model(&models.ClassroomModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("now()"))
	if tx.Error != nil {
		return fmt.Errorf("classroom delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return catalog.ErrClassroomNotFound
	}
	return nil
}

func (r *ClassroomRepo) GetByID(ctx context.Context, id uuid.UUID) (catalog.Classroom, error) {
	var m models.ClassroomModel
	err := r.dbx(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalog.Classroom{}, catalog.ErrClassroomNotFound
		}
		return catalog.Classroom{}, fmt.Errorf("classroom get: %w", err)
	}
	return models.ClassroomFromModel(m)
}

func (r *ClassroomRepo) List(ctx context.Context) ([]catalog.Classroom, error) {
	var ms []models.ClassroomModel
	err := r.dbx(ctx).
		Where("deleted_at IS NULL").
		Order("building ASC, room_number ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("classroom list: %w", err)
	}
	out := make([]catalog.Classroom, 0, len(ms))
	for _, m := range ms {
		c, err := models.ClassroomFromModel(m)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
