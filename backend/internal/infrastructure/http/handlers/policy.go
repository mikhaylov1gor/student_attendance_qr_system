package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apppolicy "attendance/internal/application/policy"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
	"attendance/internal/platform/authctx"
)

type PolicyHandler struct {
	svc *apppolicy.Service
	log *slog.Logger
}

func NewPolicyHandler(svc *apppolicy.Service, log *slog.Logger) *PolicyHandler {
	return &PolicyHandler{svc: svc, log: log}
}

func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	resp := dto.PolicyListResponse{
		Items: make([]dto.PolicyResponse, 0, len(items)),
		Total: len(items),
	}
	for _, p := range items {
		resp.Items = append(resp.Items, dto.PolicyFromDomain(p))
	}
	httperr.WriteJSON(w, http.StatusOK, resp)
}

func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.PolicyFromDomain(p))
}

func (h *PolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePolicyRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	principal, err := authctx.Require(r.Context())
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	in := apppolicy.CreateInput{
		Name:       req.Name,
		Mechanisms: req.Mechanisms,
		IsDefault:  req.IsDefault,
		CreatedBy:  &principal.UserID,
	}
	p, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusCreated, dto.PolicyFromDomain(p))
}

func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdatePolicyRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	p, err := h.svc.Update(r.Context(), id, apppolicy.UpdateInput{
		Name:       req.Name,
		Mechanisms: req.Mechanisms,
	})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.PolicyFromDomain(p))
}

func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PolicyHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.SetDefault(r.Context(), id); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseUUIDParam — читает URL-параметр как uuid, пишет 400 в случае ошибки.
func parseUUIDParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, name)
	id, err := uuid.Parse(raw)
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_id", "not a valid uuid")
		return uuid.UUID{}, false
	}
	return id, true
}
