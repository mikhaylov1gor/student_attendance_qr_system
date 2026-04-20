// Package handlers — HTTP-хендлеры.
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	appauth "attendance/internal/application/auth"
	"attendance/internal/domain/auth"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
	"attendance/internal/platform/authctx"
)

// AuthHandler держит auth-use case и логгер.
type AuthHandler struct {
	svc *appauth.Service
	log *slog.Logger
}

func NewAuthHandler(svc *appauth.Service, log *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: log}
}

// Login — POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	req.Email = dto.NormalizeEmail(req.Email)

	pair, _, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			httperr.Write(w, http.StatusUnauthorized, "invalid_credentials", "wrong email or password")
		default:
			httperr.LogUnexpected(h.log, r, err)
			httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		}
		return
	}

	httperr.WriteJSON(w, http.StatusOK, tokenResponse(pair))
}

// Refresh — POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}

	pair, _, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidToken):
			httperr.Write(w, http.StatusUnauthorized, "invalid_token", "refresh token invalid")
		case errors.Is(err, auth.ErrTokenExpired):
			httperr.Write(w, http.StatusUnauthorized, "token_expired", "refresh token expired")
		case errors.Is(err, auth.ErrTokenRevoked):
			httperr.Write(w, http.StatusUnauthorized, "token_revoked", "refresh token revoked")
		default:
			httperr.LogUnexpected(h.log, r, err)
			httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		}
		return
	}

	httperr.WriteJSON(w, http.StatusOK, tokenResponse(pair))
}

// Logout — POST /api/v1/auth/logout (требует access-токен).
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}

	if _, err := authctx.Require(r.Context()); err != nil {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "no principal")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me — GET /api/v1/auth/me (требует access-токен).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	p, err := authctx.Require(r.Context())
	if err != nil {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "no principal")
		return
	}

	u, err := h.svc.CurrentUser(r.Context(), p)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUnauthorized):
			httperr.Write(w, http.StatusUnauthorized, "unauthorized", "user no longer exists")
		default:
			httperr.LogUnexpected(h.log, r, err)
			httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		}
		return
	}

	resp := dto.MeResponse{
		ID:       u.ID.String(),
		Email:    u.Email,
		Role:     string(u.Role),
		FullName: u.FullName.String(),
	}
	if u.CurrentGroupID != nil {
		s := u.CurrentGroupID.String()
		resp.GroupID = &s
	}

	httperr.WriteJSON(w, http.StatusOK, resp)
}

func tokenResponse(p auth.TokenPair) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		ExpiresIn:    int(p.ExpiresIn.Seconds()),
		ExpiresAt:    p.AccessExpires,
		TokenType:    "Bearer",
	}
}
