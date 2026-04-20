package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"attendance/internal/domain/policy"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

type PolicyRepo struct{ db *gorm.DB }

func NewPolicyRepo(db *gorm.DB) *PolicyRepo { return &PolicyRepo{db: db} }

var _ policy.Repository = (*PolicyRepo)(nil)

func (r *PolicyRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

func (r *PolicyRepo) Create(ctx context.Context, p policy.SecurityPolicy) error {
	m, err := models.PolicyToModel(p)
	if err != nil {
		return err
	}
	if err := r.dbx(ctx).Create(&m).Error; err != nil {
		if isUniqueViolation(err, "security_policies_name_key") {
			return policy.ErrNameTaken
		}
		if isUniqueViolation(err, "uniq_default_policy") {
			return fmt.Errorf("policy create: another default policy already exists")
		}
		return fmt.Errorf("policy create: %w", err)
	}
	return nil
}

func (r *PolicyRepo) Update(ctx context.Context, p policy.SecurityPolicy) error {
	m, err := models.PolicyToModel(p)
	if err != nil {
		return err
	}
	tx := r.dbx(ctx).
		Model(&models.SecurityPolicyModel{}).
		Where("id = ? AND deleted_at IS NULL", p.ID).
		Updates(map[string]any{
			"name":       m.Name,
			"mechanisms": m.Mechanisms,
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error, "security_policies_name_key") {
			return policy.ErrNameTaken
		}
		return fmt.Errorf("policy update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return policy.ErrNotFound
	}
	return nil
}

func (r *PolicyRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	// Default-политика не может быть удалена — это инвариант уровня приложения.
	return r.dbx(ctx).Transaction(func(tx *gorm.DB) error {
		var m models.SecurityPolicyModel
		if err := tx.Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return policy.ErrNotFound
			}
			return fmt.Errorf("policy get for delete: %w", err)
		}
		if m.IsDefault {
			return policy.ErrDeletingDefault
		}
		res := tx.Model(&models.SecurityPolicyModel{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("deleted_at", gorm.Expr("now()"))
		if res.Error != nil {
			return fmt.Errorf("policy delete: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return policy.ErrNotFound
		}
		return nil
	})
}

func (r *PolicyRepo) GetByID(ctx context.Context, id uuid.UUID) (policy.SecurityPolicy, error) {
	var m models.SecurityPolicyModel
	err := r.dbx(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return policy.SecurityPolicy{}, policy.ErrNotFound
		}
		return policy.SecurityPolicy{}, fmt.Errorf("policy get: %w", err)
	}
	return models.PolicyFromModel(m)
}

func (r *PolicyRepo) GetDefault(ctx context.Context) (policy.SecurityPolicy, error) {
	var m models.SecurityPolicyModel
	err := r.dbx(ctx).
		Where("is_default = true AND deleted_at IS NULL").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return policy.SecurityPolicy{}, policy.ErrNoDefault
		}
		return policy.SecurityPolicy{}, fmt.Errorf("policy get default: %w", err)
	}
	return models.PolicyFromModel(m)
}

func (r *PolicyRepo) List(ctx context.Context) ([]policy.SecurityPolicy, error) {
	var ms []models.SecurityPolicyModel
	err := r.dbx(ctx).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("policy list: %w", err)
	}
	out := make([]policy.SecurityPolicy, 0, len(ms))
	for _, m := range ms {
		p, err := models.PolicyFromModel(m)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// SetDefault — атомарно снимает флаг со старой default-политики и ставит на
// указанную. Между этими двумя обновлениями partial-unique-index мог бы
// сработать, поэтому делаем в одной транзакции и сначала снимаем.
func (r *PolicyRepo) SetDefault(ctx context.Context, id uuid.UUID) error {
	return r.dbx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.SecurityPolicyModel{}).
			Where("id = ? AND deleted_at IS NULL", id).
			First(&models.SecurityPolicyModel{}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return policy.ErrNotFound
			}
			return fmt.Errorf("policy set default check: %w", err)
		}
		if err := tx.Model(&models.SecurityPolicyModel{}).
			Where("is_default = true AND id <> ? AND deleted_at IS NULL", id).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("policy unset default: %w", err)
		}
		if err := tx.Model(&models.SecurityPolicyModel{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("is_default", true).Error; err != nil {
			return fmt.Errorf("policy set default: %w", err)
		}
		return nil
	})
}
