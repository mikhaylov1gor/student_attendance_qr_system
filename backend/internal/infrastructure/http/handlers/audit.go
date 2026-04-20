package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain/audit"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
)

type AuditHandler struct {
	svc *appaudit.Service
	log *slog.Logger
}

func NewAuditHandler(svc *appaudit.Service, log *slog.Logger) *AuditHandler {
	return &AuditHandler{svc: svc, log: log}
}

// List — GET /api/v1/audit?limit&offset&action&actor_id&entity_type&entity_id&from&to
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := audit.ListFilter{}

	if raw := q.Get("actor_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_actor_id", "not a uuid")
			return
		}
		f.ActorID = &id
	}
	if raw := q.Get("action"); raw != "" {
		a := audit.Action(raw)
		f.Action = &a
	}
	if raw := q.Get("entity_type"); raw != "" {
		f.EntityType = &raw
	}
	if raw := q.Get("entity_id"); raw != "" {
		f.EntityID = &raw
	}
	if raw := q.Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_from", "not RFC3339")
			return
		}
		f.FromTime = &t
	}
	if raw := q.Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_to", "not RFC3339")
			return
		}
		f.ToTime = &t
	}
	f.Limit = parseIntDefault(q.Get("limit"), 50)
	f.Offset = parseIntDefault(q.Get("offset"), 0)
	if f.Limit > 500 {
		f.Limit = 500
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	resp := dto.AuditListResponse{
		Items: make([]dto.AuditEntryResponse, 0, len(items)),
		Total: total,
	}
	for _, e := range items {
		resp.Items = append(resp.Items, dto.AuditEntryFromDomain(e))
	}
	httperr.WriteJSON(w, http.StatusOK, resp)
}

// Verify — POST /api/v1/audit/verify (тело не требуется).
func (h *AuditHandler) Verify(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Verify(r.Context())
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.AuditVerifyResponse{
		OK:            res.OK,
		TotalEntries:  res.TotalEntries,
		FirstBrokenID: res.FirstBrokenID,
		BrokenReason:  res.BrokenReason,
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}
