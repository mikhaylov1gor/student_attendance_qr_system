package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"attendance/internal/domain/auth"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

type RefreshTokenRepo struct{ db *gorm.DB }

func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo { return &RefreshTokenRepo{db: db} }

var _ auth.RefreshTokenRepository = (*RefreshTokenRepo)(nil)

func (r *RefreshTokenRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

func (r *RefreshTokenRepo) Store(ctx context.Context, t auth.RefreshToken) error {
	m := models.RefreshTokenModel{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		IssuedAt:  t.IssuedAt,
		ExpiresAt: t.ExpiresAt,
		RevokedAt: t.RevokedAt,
	}
	if err := r.dbx(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("refresh token store: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByHash(ctx context.Context, hash []byte) (auth.RefreshToken, error) {
	var m models.RefreshTokenModel
	err := r.dbx(ctx).
		Where("token_hash = ?", hash).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return auth.RefreshToken{}, auth.ErrInvalidToken
		}
		return auth.RefreshToken{}, fmt.Errorf("refresh token get: %w", err)
	}
	return auth.RefreshToken{
		ID:        m.ID,
		UserID:    m.UserID,
		TokenHash: m.TokenHash,
		IssuedAt:  m.IssuedAt,
		ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt,
	}, nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	if err := r.dbx(ctx).
		Model(&models.RefreshTokenModel{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", gorm.Expr("now()")).Error; err != nil {
		return fmt.Errorf("refresh token revoke: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if err := r.dbx(ctx).
		Model(&models.RefreshTokenModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", gorm.Expr("now()")).Error; err != nil {
		return fmt.Errorf("refresh token revoke-all: %w", err)
	}
	return nil
}
