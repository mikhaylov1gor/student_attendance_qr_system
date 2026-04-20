package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"attendance/internal/domain"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

// UserRepo — реализация user.Repository поверх Gorm + AES-GCM-шифрование ФИО.
type UserRepo struct {
	db  *gorm.DB
	enc domain.FieldEncryptor
}

func NewUserRepo(db *gorm.DB, enc domain.FieldEncryptor) *UserRepo {
	return &UserRepo{db: db, enc: enc}
}

var _ user.Repository = (*UserRepo)(nil)

// dbx — tx-aware доступ к БД: внутри TxRunner.Run возвращает текущую транзакцию.
func (r *UserRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

func (r *UserRepo) Create(ctx context.Context, u user.User) error {
	m, err := models.UserToModel(u, r.enc)
	if err != nil {
		return err
	}
	if err := r.dbx(ctx).Create(m).Error; err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return user.ErrEmailTaken
		}
		return fmt.Errorf("user create: %w", err)
	}
	return nil
}

func (r *UserRepo) Update(ctx context.Context, u user.User) error {
	m, err := models.UserToModel(u, r.enc)
	if err != nil {
		return err
	}
	tx := r.dbx(ctx).
		Model(&models.UserModel{}).
		Where("id = ? AND deleted_at IS NULL", u.ID).
		Updates(map[string]any{
			"email":                m.Email,
			"password_hash":        m.PasswordHash,
			"full_name_ciphertext": m.FullNameCiphertext,
			"full_name_nonce":      m.FullNameNonce,
			"role":                 m.Role,
			"current_group_id":     m.CurrentGroupID,
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error, "users_email_key") {
			return user.ErrEmailTaken
		}
		return fmt.Errorf("user update: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *UserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tx := r.dbx(ctx).
		Model(&models.UserModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("now()"))
	if tx.Error != nil {
		return fmt.Errorf("user soft delete: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	var m models.UserModel
	err := r.dbx(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, fmt.Errorf("user get by id: %w", err)
	}
	return models.UserFromModel(m, r.enc)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (user.User, error) {
	var m models.UserModel
	err := r.dbx(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, fmt.Errorf("user get by email: %w", err)
	}
	return models.UserFromModel(m, r.enc)
}

func (r *UserRepo) List(ctx context.Context, f user.ListFilter) ([]user.User, int, error) {
	q := r.dbx(ctx).Model(&models.UserModel{}).Where("deleted_at IS NULL")
	if f.Role != nil {
		q = q.Where("role = ?", string(*f.Role))
	}
	if f.GroupID != nil {
		q = q.Where("current_group_id = ?", *f.GroupID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("user list count: %w", err)
	}

	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	var ms []models.UserModel
	if err := q.Order("created_at ASC").Find(&ms).Error; err != nil {
		return nil, 0, fmt.Errorf("user list: %w", err)
	}

	out := make([]user.User, 0, len(ms))
	for _, m := range ms {
		u, err := models.UserFromModel(m, r.enc)
		if err != nil {
			return nil, 0, fmt.Errorf("decrypt user %s: %w", m.ID, err)
		}
		out = append(out, u)
	}
	return out, int(total), nil
}
