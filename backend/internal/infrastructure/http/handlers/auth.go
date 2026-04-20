// Package handlers — HTTP-хендлеры.
package handlers

import (
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
		httperr.RespondError(w, r, h.log, err)
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
		httperr.RespondError(w, r, h.log, err)
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
		httperr.RespondError(w, r, h.log, err)
		return
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me — GET /api/v1/auth/me (требует access-токен).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	p, err := authctx.Require(r.Context())
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}

	u, err := h.svc.CurrentUser(r.Context(), p)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
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
