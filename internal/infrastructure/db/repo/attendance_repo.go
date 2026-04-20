package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"attendance/internal/domain/attendance"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

type AttendanceRepo struct{ db *gorm.DB }

func NewAttendanceRepo(db *gorm.DB) *AttendanceRepo { return &AttendanceRepo{db: db} }

var _ attendance.Repository = (*AttendanceRepo)(nil)

func (r *AttendanceRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

// Submit пишет одну запись отметки + N записей результатов проверок.
// Должен вызываться из TxRunner.Run, чтобы стать частью общей транзакции
// (вместе с audit.Append).
//
// Уникальный индекс uniq_attendance_session_student обеспечивает идемпотентность:
// конкурирующая попытка вставить дубликат ломается на 23505 → ErrAlreadySubmitted.
func (r *AttendanceRepo) Submit(ctx context.Context, rec attendance.Record, checks []attendance.CheckResult) error {
	mr := models.AttendanceRecordToModel(rec)
	if err := r.dbx(ctx).Create(&mr).Error; err != nil {
		if isUniqueViolation(err, "uniq_attendance_session_student") {
			return attendance.ErrAlreadySubmitted
		}
		return fmt.Errorf("attendance submit: insert record: %w", err)
	}
	if len(checks) == 0 {
		return nil
	}
	mchecks := make([]models.SecurityCheckResultModel, 0, len(checks))
	for _, c := range checks {
		c.AttendanceID = rec.ID
		m, err := models.CheckResultToModel(c)
		if err != nil {
			return fmt.Errorf("attendance submit: map check: %w", err)
		}
		mchecks = append(mchecks, m)
	}
	if err := r.dbx(ctx).Create(&mchecks).Error; err != nil {
		return fmt.Errorf("attendance submit: insert checks: %w", err)
	}
	return nil
}

// Resolve проставляет final_status, resolved_by, resolved_at + notes.
// Применяется только к записям, где final_status пока NULL (защита от
// гонки двух teacher'ов одновременно).
func (r *AttendanceRepo) Resolve(
	ctx context.Context,
	id uuid.UUID,
	finalStatus attendance.Status,
	resolvedBy uuid.UUID,
	notes string,
) error {
	if !finalStatus.IsValidFinal() {
		return attendance.ErrInvalidFinal
	}
	tx := r.dbx(ctx).
		Model(&models.AttendanceRecordModel{}).
		Where("id = ? AND final_status IS NULL", id).
		Updates(map[string]any{
			"final_status": string(finalStatus),
			"resolved_by":  resolvedBy,
			"resolved_at":  gorm.Expr("now()"),
			"notes":        notes,
		})
	if tx.Error != nil {
		return fmt.Errorf("attendance resolve: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return attendance.ErrNotResolvable
	}
	return nil
}

func (r *AttendanceRepo) GetByID(ctx context.Context, id uuid.UUID) (attendance.Record, []attendance.CheckResult, error) {
	var mr models.AttendanceRecordModel
	err := r.dbx(ctx).Where("id = ?", id).First(&mr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendance.Record{}, nil, attendance.ErrNotFound
		}
		return attendance.Record{}, nil, fmt.Errorf("attendance get: %w", err)
	}
	rec := models.AttendanceRecordFromModel(mr)

	var mchecks []models.SecurityCheckResultModel
	if err := r.dbx(ctx).
		Where("attendance_id = ?", id).
		Order("checked_at ASC").
		Find(&mchecks).Error; err != nil {
		return attendance.Record{}, nil, fmt.Errorf("attendance get checks: %w", err)
	}
	checks := make([]attendance.CheckResult, 0, len(mchecks))
	for _, m := range mchecks {
		c, err := models.CheckResultFromModel(m)
		if err != nil {
			return attendance.Record{}, nil, err
		}
		checks = append(checks, c)
	}
	return rec, checks, nil
}

func (r *AttendanceRepo) ExistsForSessionStudent(ctx context.Context, sessionID, studentID uuid.UUID) (bool, error) {
	var count int64
	err := r.dbx(ctx).
		Model(&models.AttendanceRecordModel{}).
		Where("session_id = ? AND student_id = ?", sessionID, studentID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("attendance exists: %w", err)
	}
	return count > 0, nil
}

func (r *AttendanceRepo) List(ctx context.Context, f attendance.ListFilter) ([]attendance.Record, int, error) {
	q := r.dbx(ctx).Model(&models.AttendanceRecordModel{})
	if f.SessionID != nil {
		q = q.Where("session_id = ?", *f.SessionID)
	}
	if f.StudentID != nil {
		q = q.Where("student_id = ?", *f.StudentID)
	}
	if f.FromTime != nil {
		q = q.Where("submitted_at >= ?", *f.FromTime)
	}
	if f.ToTime != nil {
		q = q.Where("submitted_at <= ?", *f.ToTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("attendance list count: %w", err)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	var ms []models.AttendanceRecordModel
	if err := q.Order("submitted_at DESC").Find(&ms).Error; err != nil {
		return nil, 0, fmt.Errorf("attendance list: %w", err)
	}
	out := make([]attendance.Record, 0, len(ms))
	for _, m := range ms {
		out = append(out, models.AttendanceRecordFromModel(m))
	}
	return out, int(total), nil
}
