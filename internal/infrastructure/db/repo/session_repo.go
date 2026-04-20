package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"attendance/internal/domain/session"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

// SessionRepo — реализация session.Repository. M:N состав групп хранится в
// отдельной таблице session_groups и загружается/сохраняется вместе с сессией.
type SessionRepo struct{ db *gorm.DB }

func NewSessionRepo(db *gorm.DB) *SessionRepo { return &SessionRepo{db: db} }

var _ session.Repository = (*SessionRepo)(nil)

func (r *SessionRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

// Create пишет сессию и её состав групп в одной транзакции.
func (r *SessionRepo) Create(ctx context.Context, s session.Session) error {
	return r.dbx(ctx).Transaction(func(tx *gorm.DB) error {
		m := models.SessionToModel(s)
		if err := tx.Create(&m).Error; err != nil {
			return fmt.Errorf("session create: %w", err)
		}
		if err := r.replaceGroups(tx, s.ID, s.GroupIDs); err != nil {
			return err
		}
		return nil
	})
}

// Update изменяет метаданные сессии и пересобирает состав групп.
func (r *SessionRepo) Update(ctx context.Context, s session.Session) error {
	return r.dbx(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.SessionModel{}).Where("id = ?", s.ID).Updates(map[string]any{
			"teacher_id":         s.TeacherID,
			"course_id":          s.CourseID,
			"classroom_id":       s.ClassroomID,
			"security_policy_id": s.SecurityPolicyID,
			"starts_at":          s.StartsAt,
			"ends_at":            s.EndsAt,
			"status":             string(s.Status),
			"qr_secret":          s.QRSecret,
			"qr_ttl_seconds":     s.QRTTLSeconds,
			"qr_counter":         s.QRCounter,
		})
		if res.Error != nil {
			return fmt.Errorf("session update: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return session.ErrNotFound
		}
		if err := r.replaceGroups(tx, s.ID, s.GroupIDs); err != nil {
			return err
		}
		return nil
	})
}

func (r *SessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.dbx(ctx).Delete(&models.SessionModel{}, "id = ?", id)
	if tx.Error != nil {
		return fmt.Errorf("session delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return session.ErrNotFound
	}
	return nil
}

func (r *SessionRepo) GetByID(ctx context.Context, id uuid.UUID) (session.Session, error) {
	var m models.SessionModel
	err := r.dbx(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return session.Session{}, session.ErrNotFound
		}
		return session.Session{}, fmt.Errorf("session get: %w", err)
	}
	groups, err := r.groupsOf(ctx, id)
	if err != nil {
		return session.Session{}, err
	}
	return models.SessionFromModel(m, groups), nil
}

func (r *SessionRepo) List(ctx context.Context, f session.ListFilter) ([]session.Session, int, error) {
	q := r.dbx(ctx).Model(&models.SessionModel{})
	if f.TeacherID != nil {
		q = q.Where("teacher_id = ?", *f.TeacherID)
	}
	if f.CourseID != nil {
		q = q.Where("course_id = ?", *f.CourseID)
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.FromTime != nil {
		q = q.Where("starts_at >= ?", *f.FromTime)
	}
	if f.ToTime != nil {
		q = q.Where("ends_at <= ?", *f.ToTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("session list count: %w", err)
	}

	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	var ms []models.SessionModel
	if err := q.Order("starts_at DESC").Find(&ms).Error; err != nil {
		return nil, 0, fmt.Errorf("session list: %w", err)
	}

	// Загружаем группы для каждой сессии. N+1, но для типичных объёмов (десятки
	// сессий на страницу) приемлемо. Если станет узким местом — IN-запрос.
	out := make([]session.Session, 0, len(ms))
	for _, m := range ms {
		groups, err := r.groupsOf(ctx, m.ID)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, models.SessionFromModel(m, groups))
	}
	return out, int(total), nil
}

func (r *SessionRepo) ActiveForBootstrap(ctx context.Context) ([]session.Session, error) {
	var ms []models.SessionModel
	err := r.dbx(ctx).Where("status = ?", string(session.StatusActive)).Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("session active bootstrap: %w", err)
	}
	out := make([]session.Session, 0, len(ms))
	for _, m := range ms {
		groups, err := r.groupsOf(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, models.SessionFromModel(m, groups))
	}
	return out, nil
}

// IncrementQRCounter атомарно увеличивает qr_counter на 1 и возвращает новое
// значение. RETURNING qr_counter используется Postgres — Gorm умеет его
// пробрасывать через Clauses.
func (r *SessionRepo) IncrementQRCounter(ctx context.Context, id uuid.UUID) (int, error) {
	var next int
	err := r.dbx(ctx).Raw(`
		UPDATE sessions SET qr_counter = qr_counter + 1
		WHERE id = ? AND status = 'active'
		RETURNING qr_counter`, id).Scan(&next).Error
	if err != nil {
		return 0, fmt.Errorf("session inc counter: %w", err)
	}
	if next == 0 {
		// Sanity: RETURNING вернёт как минимум 1 после инкремента;
		// 0 значит, что UPDATE не зацепил ни одну строку.
		return 0, session.ErrNotFound
	}
	return next, nil
}

// replaceGroups — удаляет текущий состав session_groups и вставляет новый
// (для Create можно было бы ограничиться вставкой, но унифицируем через один метод).
func (r *SessionRepo) replaceGroups(tx *gorm.DB, sessionID uuid.UUID, groups []uuid.UUID) error {
	if err := tx.
		Where("session_id = ?", sessionID).
		Delete(&models.SessionGroupModel{}).Error; err != nil {
		return fmt.Errorf("session groups clear: %w", err)
	}
	if len(groups) == 0 {
		return nil
	}
	rows := make([]models.SessionGroupModel, len(groups))
	for i, gid := range groups {
		rows[i] = models.SessionGroupModel{SessionID: sessionID, GroupID: gid}
	}
	if err := tx.Create(&rows).Error; err != nil {
		return fmt.Errorf("session groups insert: %w", err)
	}
	return nil
}

func (r *SessionRepo) groupsOf(ctx context.Context, sessionID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.dbx(ctx).
		Table("session_groups").
		Where("session_id = ?", sessionID).
		Order("group_id").
		Pluck("group_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("session groups load: %w", err)
	}
	return ids, nil
}
