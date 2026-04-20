// Package auth — use case'ы аутентификации: Login / Refresh / Logout / CurrentUser.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain"
	"attendance/internal/domain/audit"
	"attendance/internal/domain/auth"
	"attendance/internal/domain/user"
	"attendance/internal/platform/requestmeta"
)

// Deps — зависимости use case'а. Все тесты подкладывают фейки сюда.
type Deps struct {
	Users      user.Repository
	Tokens     auth.RefreshTokenRepository // хранилище refresh-токенов
	Hasher     domain.PasswordHasher
	Signer     auth.AccessTokenSigner
	Clock      domain.Clock
	Tx         domain.TxRunner // транзакция: основная мутация + audit.Append
	Audit      *appaudit.Service
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Service собирает use case'ы в один объект.
type Service struct{ Deps }

func NewService(d Deps) *Service {
	if d.AccessTTL <= 0 {
		d.AccessTTL = 15 * time.Minute
	}
	if d.RefreshTTL <= 0 {
		d.RefreshTTL = 7 * 24 * time.Hour
	}
	return &Service{Deps: d}
}

// Login — email+password → пара токенов. Неправильная пара email/password
// возвращает ErrInvalidCredentials без раскрытия, что именно не совпало.
//
// Успешный и неуспешный логин аудитятся в той же транзакции, что выдача
// refresh-токена (audit и store — одна tx).
func (s *Service) Login(ctx context.Context, email, password string) (auth.TokenPair, auth.Principal, error) {
	u, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			s.auditLoginFailed(ctx, email, "user_not_found")
			return auth.TokenPair{}, auth.Principal{}, auth.ErrInvalidCredentials
		}
		return auth.TokenPair{}, auth.Principal{}, fmt.Errorf("login: load user: %w", err)
	}

	ok, err := s.Hasher.Verify(password, u.PasswordHash)
	if err != nil || !ok {
		s.auditLoginFailed(ctx, email, "invalid_password")
		return auth.TokenPair{}, auth.Principal{}, auth.ErrInvalidCredentials
	}

	var pair auth.TokenPair
	err = s.Tx.Run(ctx, func(txCtx context.Context) error {
		var innerErr error
		pair, innerErr = s.issuePair(txCtx, u)
		if innerErr != nil {
			return innerErr
		}
		return s.auditAppend(txCtx, audit.Entry{
			ActorID:    &u.ID,
			ActorRole:  string(u.Role),
			Action:     audit.ActionLoginSuccess,
			EntityType: "user",
			EntityID:   u.ID.String(),
			Payload: map[string]any{
				"email": u.Email,
			},
		})
	})
	if err != nil {
		return auth.TokenPair{}, auth.Principal{}, err
	}
	return pair, auth.Principal{UserID: u.ID, Role: u.Role}, nil
}

// Refresh — ротация refresh-токена: проверяем, отзываем, выдаём новую пару.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (auth.TokenPair, auth.Principal, error) {
	hash := sha256Sum([]byte(refreshToken))

	existing, err := s.Tokens.GetByHash(ctx, hash)
	if err != nil {
		return auth.TokenPair{}, auth.Principal{}, err
	}

	now := s.Clock.Now(ctx)
	if existing.RevokedAt != nil {
		return auth.TokenPair{}, auth.Principal{}, auth.ErrTokenRevoked
	}
	if now.After(existing.ExpiresAt) {
		return auth.TokenPair{}, auth.Principal{}, auth.ErrTokenExpired
	}

	u, err := s.Users.GetByID(ctx, existing.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// Пользователь удалён — токен невалиден.
			return auth.TokenPair{}, auth.Principal{}, auth.ErrInvalidToken
		}
		return auth.TokenPair{}, auth.Principal{}, fmt.Errorf("refresh: load user: %w", err)
	}

	var pair auth.TokenPair
	err = s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Tokens.Revoke(txCtx, existing.ID); err != nil {
			return fmt.Errorf("refresh: revoke old: %w", err)
		}
		var innerErr error
		pair, innerErr = s.issuePair(txCtx, u)
		if innerErr != nil {
			return innerErr
		}
		return s.auditAppend(txCtx, audit.Entry{
			ActorID:    &u.ID,
			ActorRole:  string(u.Role),
			Action:     audit.ActionTokenRefreshed,
			EntityType: "user",
			EntityID:   u.ID.String(),
			Payload: map[string]any{
				"revoked_token_id": existing.ID.String(),
			},
		})
	})
	if err != nil {
		return auth.TokenPair{}, auth.Principal{}, err
	}
	return pair, auth.Principal{UserID: u.ID, Role: u.Role}, nil
}

// Logout — отзыв конкретного refresh-токена. Если токен уже отозван или
// не найден — возвращаем nil (idempotent, audit не пишем).
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := sha256Sum([]byte(refreshToken))
	existing, err := s.Tokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil
		}
		return err
	}
	if existing.RevokedAt != nil {
		return nil
	}
	return s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Tokens.Revoke(txCtx, existing.ID); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			ActorID:    &existing.UserID,
			Action:     audit.ActionLogout,
			EntityType: "user",
			EntityID:   existing.UserID.String(),
			Payload: map[string]any{
				"token_id": existing.ID.String(),
			},
		})
	})
}

// CurrentUser — вернуть текущего пользователя по Principal из ctx.
func (s *Service) CurrentUser(ctx context.Context, p auth.Principal) (user.User, error) {
	u, err := s.Users.GetByID(ctx, p.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// principal из токена указывает на удалённого пользователя
			return user.User{}, auth.ErrUnauthorized
		}
		return user.User{}, err
	}
	return u, nil
}

// issuePair — общий путь выдачи пары для Login и Refresh.
func (s *Service) issuePair(ctx context.Context, u user.User) (auth.TokenPair, error) {
	now := s.Clock.Now(ctx)
	accessExp := now.Add(s.AccessTTL)
	refreshExp := now.Add(s.RefreshTTL)

	accessJTI := uuid.New()
	access, err := s.Signer.Sign(auth.AccessClaims{
		Subject:  u.ID,
		Role:     u.Role,
		IssuedAt: now,
		Expires:  accessExp,
		TokenID:  accessJTI,
	})
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("issue access: %w", err)
	}

	refresh, err := generateRefreshToken()
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("generate refresh: %w", err)
	}
	refreshHash := sha256Sum([]byte(refresh))

	err = s.Tokens.Store(ctx, auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: refreshHash,
		IssuedAt:  now,
		ExpiresAt: refreshExp,
	})
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("store refresh: %w", err)
	}

	return auth.TokenPair{
		AccessToken:    access,
		RefreshToken:   refresh,
		ExpiresIn:      s.AccessTTL,
		AccessExpires:  accessExp,
		RefreshExpires: refreshExp,
	}, nil
}

// generateRefreshToken — 32 байта из crypto/rand, закодированные base64url.
func generateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

// auditAppend — helper: дополняет entry request-метаданными из ctx и вызывает
// Audit.Append. Ошибка прокидывается наверх — транзакция откатится.
func (s *Service) auditAppend(ctx context.Context, e audit.Entry) error {
	if s.Audit == nil {
		return nil
	}
	meta := requestmeta.From(ctx)
	e.IPAddress = meta.RemoteIP
	e.UserAgent = meta.UserAgent
	_, err := s.Audit.Append(ctx, e)
	return err
}

// auditLoginFailed — отдельная ветка: не внутри tx (т.к. нет основной мутации),
// но тоже пишем в журнал. Ошибку просто проглатываем — не хотим давать
// атакующему подсказки через response code.
func (s *Service) auditLoginFailed(ctx context.Context, email, reason string) {
	if s.Audit == nil {
		return
	}
	meta := requestmeta.From(ctx)
	_, _ = s.Audit.Append(ctx, audit.Entry{
		Action:     audit.ActionLoginFailed,
		EntityType: "user",
		EntityID:   email,
		Payload: map[string]any{
			"email":  email,
			"reason": reason,
		},
		IPAddress: meta.RemoteIP,
		UserAgent: meta.UserAgent,
	})
}

// ensure uuid import stays used if last ref is removed elsewhere
var _ = uuid.Nil
