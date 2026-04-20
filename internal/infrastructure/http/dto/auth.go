package dto

import "time"

// LoginRequest — вход по email/password.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,min=1,max=512"`
}

// RefreshRequest — тело POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=16,max=512"`
}

// LogoutRequest — тело POST /auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=16,max=512"`
}

// TokenResponse — тело успешного /auth/login и /auth/refresh.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"` // секунд до истечения access
	ExpiresAt    time.Time `json:"expires_at"` // RFC3339 UTC
	TokenType    string    `json:"token_type"` // всегда "Bearer"
}

// MeResponse — тело GET /auth/me.
type MeResponse struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	FullName string  `json:"full_name"`
	GroupID  *string `json:"current_group_id,omitempty"`
}
