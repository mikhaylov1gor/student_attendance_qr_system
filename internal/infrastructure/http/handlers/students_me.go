package handlers

import (
	"log/slog"
	"net/http"

	appattendance "attendance/internal/application/attendance"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
	"attendance/internal/platform/authctx"
)

// StudentMeHandler — self-service endpoint'ы для текущего студента.
// Идентификация через Principal из JWT.
type StudentMeHandler struct {
	svc *appattendance.Service
	log *slog.Logger
}

func NewStudentMeHandler(svc *appattendance.Service, log *slog.Logger) *StudentMeHandler {
	return &StudentMeHandler{svc: svc, log: log}
}

// Attendance — GET /api/v1/students/me/attendance?limit=50&offset=0
func (h *StudentMeHandler) Attendance(w http.ResponseWriter, r *http.Request) {
	p, err := authctx.Require(r.Context())
	if err != nil {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "no principal")
		return
	}
	q := r.URL.Query()
	limit := parseIntOr(q.Get("limit"), 50)
	offset := parseIntOr(q.Get("offset"), 0)
	if limit > 500 {
		limit = 500
	}

	items, total, err := h.svc.ListMyAttendance(r.Context(), p.UserID, limit, offset)
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := dto.MyAttendanceListResponse{Total: total, Items: make([]dto.MyAttendanceItem, 0, len(items))}
	for _, it := range items {
		out.Items = append(out.Items, dto.MyAttendanceFromDomain(it))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

// Stats — GET /api/v1/students/me/stats
func (h *StudentMeHandler) Stats(w http.ResponseWriter, r *http.Request) {
	p, err := authctx.Require(r.Context())
	if err != nil {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "no principal")
		return
	}
	st, err := h.svc.GetMyStats(r.Context(), p.UserID)
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.MyStatsResponse{
		Total:          st.Total,
		Accepted:       st.Accepted,
		NeedsReview:    st.NeedsReview,
		Rejected:       st.Rejected,
		AttendanceRate: st.AttendanceRate,
	})
}
