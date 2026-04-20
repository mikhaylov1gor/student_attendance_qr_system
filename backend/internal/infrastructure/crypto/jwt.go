package crypto

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"attendance/internal/domain/auth"
	"attendance/internal/domain/user"
)

// JWTSigner реализует auth.AccessTokenSigner на HS256.
type JWTSigner struct {
	secret []byte
	issuer string
}

// NewJWTSigner: secret — сырые байты (≥32), чтобы HS256 был стойким.
func NewJWTSigner(secret []byte, issuer string) (*JWTSigner, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt: secret must be at least 32 bytes, got %d", len(secret))
	}
	return &JWTSigner{secret: secret, issuer: issuer}, nil
}

// NewJWTSignerFromBase64 — удобно для .env (JWT_ACCESS_SECRET в base64).
func NewJWTSignerFromBase64(b64 string, issuer string) (*JWTSigner, error) {
	secret, err := decodeBase64(b64)
	if err != nil {
		return nil, fmt.Errorf("jwt: decode secret: %w", err)
	}
	return NewJWTSigner(secret, issuer)
}

// Sign выписывает JWT с claims.
func (s *JWTSigner) Sign(c auth.AccessClaims) (string, error) {
	claims := jwt.MapClaims{
		"sub":  c.Subject.String(),
		"role": string(c.Role),
		"iat":  c.IssuedAt.Unix(),
		"exp":  c.Expires.Unix(),
		"jti":  c.TokenID.String(),
	}
	if s.issuer != "" {
		claims["iss"] = s.issuer
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	return signed, nil
}

// Verify парсит и проверяет подпись JWT.
// Маппит stdlib jwt errors → доменные auth.ErrXxx.
func (s *JWTSigner) Verify(token string) (auth.AccessClaims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return auth.AccessClaims{}, auth.ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed),
			errors.Is(err, jwt.ErrSignatureInvalid),
			errors.Is(err, jwt.ErrTokenNotValidYet),
			errors.Is(err, jwt.ErrTokenUsedBeforeIssued):
			return auth.AccessClaims{}, auth.ErrInvalidToken
		default:
			return auth.AccessClaims{}, auth.ErrInvalidToken
		}
	}

	mc, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return auth.AccessClaims{}, auth.ErrInvalidToken
	}

	sub, err := uuid.Parse(asString(mc["sub"]))
	if err != nil {
		return auth.AccessClaims{}, auth.ErrInvalidToken
	}
	role := user.Role(asString(mc["role"]))
	if !role.Valid() {
		return auth.AccessClaims{}, auth.ErrInvalidToken
	}

	iat := int64FromClaim(mc["iat"])
	exp := int64FromClaim(mc["exp"])
	if iat == 0 || exp == 0 {
		return auth.AccessClaims{}, auth.ErrInvalidToken
	}

	jti, err := uuid.Parse(asString(mc["jti"]))
	if err != nil {
		return auth.AccessClaims{}, auth.ErrInvalidToken
	}

	return auth.AccessClaims{
		Subject:  sub,
		Role:     role,
		IssuedAt: time.Unix(iat, 0).UTC(),
		Expires:  time.Unix(exp, 0).UTC(),
		TokenID:  jti,
	}, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func int64FromClaim(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	}
	return 0
}
