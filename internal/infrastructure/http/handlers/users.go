package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	appuser "attendance/internal/application/user"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
)

type UserHandler struct {
	svc *appuser.Service
	log *slog.Logger
}

func NewUserHandler(svc *appuser.Service, log *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, log: log}
}

// =========================================================================
// Create
// =========================================================================

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}

	fn, err := user.NewFullName(req.Last, req.First, req.Middle)
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_full_name", err.Error())
		return
	}

	out, err := h.svc.Create(r.Context(), appuser.CreateInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: fn,
		Role:     user.Role(req.Role),
		GroupID:  req.GroupID,
	})
	if err != nil {
		h.writeUserError(w, r, err)
		return
	}

	httperr.WriteJSON(w, http.StatusCreated, dto.CreateUserResponse{
		User:         dto.UserFromDomain(out.User),
		TempPassword: out.TempPassword,
	})
}

// =========================================================================
// List / Get
// =========================================================================

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := appuser.ListFilter{
		Query:  q.Get("q"),
		Limit:  parseIntOr(q.Get("limit"), 50),
		Offset: parseIntOr(q.Get("offset"), 0),
	}
	if raw := q.Get("role"); raw != "" {
		rl := user.Role(raw)
		if !rl.Valid() {
			httperr.Write(w, http.StatusBadRequest, "invalid_role", "bad role")
			return
		}
		f.Role = &rl
	}
	if raw := q.Get("group_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_group_id", "not uuid")
			return
		}
		f.GroupID = &id
	}
	items, total, err := h.svc.ListWithSearch(r.Context(), f)
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := dto.UserListResponse{Total: total, Items: make([]dto.UserResponse, 0, len(items))}
	for _, u := range items {
		out.Items = append(out.Items, dto.UserFromDomain(u))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.writeUserError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.UserFromDomain(u))
}

// =========================================================================
// Update
// =========================================================================

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateUserRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}

	// ФИО обновляется целиком (Last/First/Middle). Если любой из трёх прислан,
	// берём текущий пользователь и пересобираем на основе нового + имеющегося.
	in := appuser.UpdateInput{
		Email:      req.Email,
		GroupID:    req.GroupID,
		ClearGroup: req.ClearGroup,
	}
	if req.Role != nil {
		rl := user.Role(*req.Role)
		in.Role = &rl
	}
	if req.Last != nil || req.First != nil || req.Middle != nil {
		cur, err := h.svc.GetByID(r.Context(), id)
		if err != nil {
			h.writeUserError(w, r, err)
			return
		}
		last, first, middle := cur.FullName.Last, cur.FullName.First, cur.FullName.Middle
		if req.Last != nil {
			last = *req.Last
		}
		if req.First != nil {
			first = *req.First
		}
		if req.Middle != nil {
			middle = *req.Middle
		}
		fn, err := user.NewFullName(last, first, middle)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_full_name", err.Error())
			return
		}
		in.FullName = &fn
	}

	u, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		h.writeUserError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.UserFromDomain(u))
}

// =========================================================================
// Delete
// =========================================================================

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.writeUserError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// =========================================================================
// Reset password
// =========================================================================

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	temp, err := h.svc.ResetPassword(r.Context(), id)
	if err != nil {
		h.writeUserError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.ResetPasswordResponse{TempPassword: temp})
}

// =========================================================================
// error mapping
// =========================================================================

func (h *UserHandler) writeUserError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, user.ErrNotFound):
		httperr.Write(w, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, user.ErrEmailTaken):
		httperr.Write(w, http.StatusConflict, "email_taken", "email already in use")
	case errors.Is(err, user.ErrInvalidRole):
		httperr.Write(w, http.StatusBadRequest, "invalid_role", err.Error())
	case errors.Is(err, user.ErrRoleGroupMismatch):
		httperr.Write(w, http.StatusBadRequest, "role_group_mismatch",
			"current_group_id is required for students and forbidden for other roles")
	case errors.Is(err, user.ErrFullNameRequired):
		httperr.Write(w, http.StatusBadRequest, "invalid_full_name", err.Error())
	case errors.Is(err, user.ErrFullNameTooLong):
		httperr.Write(w, http.StatusBadRequest, "invalid_full_name", err.Error())
	default:
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}
