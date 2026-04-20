package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	appsession "attendance/internal/application/session"
	"attendance/internal/domain/catalog"
	"attendance/internal/domain/policy"
	"attendance/internal/domain/session"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
)

type SessionHandler struct {
	svc *appsession.Service
	log *slog.Logger
}

func NewSessionHandler(svc *appsession.Service, log *slog.Logger) *SessionHandler {
	return &SessionHandler{svc: svc, log: log}
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateSessionRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	s, err := h.svc.CreateDraft(r.Context(), appsession.CreateInput{
		CourseID:         req.CourseID,
		ClassroomID:      req.ClassroomID,
		SecurityPolicyID: req.SecurityPolicyID,
		StartsAt:         req.StartsAt,
		EndsAt:           req.EndsAt,
		GroupIDs:         req.GroupIDs,
		QRTTLSeconds:     req.QRTTLSeconds,
	})
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusCreated, dto.SessionFromDomain(s))
}

func (h *SessionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateSessionRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}

	// ClassroomID: если пришёл clear_classroom=true — значит снять привязку.
	// Иначе если ClassroomID пришёл — установить его. Иначе не трогать.
	var classroom *appsession.OptUUID
	if req.ClearClassroom {
		classroom = &appsession.OptUUID{Set: true, Value: nil}
	} else if req.ClassroomID != nil {
		classroom = &appsession.OptUUID{Set: true, Value: req.ClassroomID}
	}

	s, err := h.svc.Update(r.Context(), id, appsession.UpdateInput{
		ClassroomID:      classroom,
		SecurityPolicyID: req.SecurityPolicyID,
		StartsAt:         req.StartsAt,
		EndsAt:           req.EndsAt,
		GroupIDs:         req.GroupIDs,
		QRTTLSeconds:     req.QRTTLSeconds,
	})
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.SessionFromDomain(s))
}

func (h *SessionHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	s, err := h.svc.Start(r.Context(), id)
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.SessionFromDomain(s))
}

func (h *SessionHandler) Close(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	s, err := h.svc.Close(r.Context(), id)
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.SessionFromDomain(s))
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	s, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.SessionFromDomain(s))
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := session.ListFilter{Limit: 50}

	if raw := q.Get("teacher_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_teacher_id", "not uuid")
			return
		}
		f.TeacherID = &id
	}
	if raw := q.Get("course_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_course_id", "not uuid")
			return
		}
		f.CourseID = &id
	}
	if raw := q.Get("status"); raw != "" {
		st := session.Status(raw)
		if !st.Valid() {
			httperr.Write(w, http.StatusBadRequest, "invalid_status", "bad session status")
			return
		}
		f.Status = &st
	}
	if raw := q.Get("from"); raw != "" {
		if _, err := time.Parse(time.RFC3339, raw); err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_from", "not RFC3339")
			return
		}
		f.FromTime = &raw
	}
	if raw := q.Get("to"); raw != "" {
		if _, err := time.Parse(time.RFC3339, raw); err != nil {
			httperr.Write(w, http.StatusBadRequest, "invalid_to", "not RFC3339")
			return
		}
		f.ToTime = &raw
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := dto.SessionListResponse{Total: total, Items: make([]dto.SessionResponse, 0, len(items))}
	for _, s := range items {
		out.Items = append(out.Items, dto.SessionFromDomain(s))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

// Attendance — заготовка для stage 9. Сейчас возвращает 501.
func (h *SessionHandler) Attendance(w http.ResponseWriter, r *http.Request) {
	httperr.Write(w, http.StatusNotImplemented, "not_implemented", "attendance endpoints will be wired in stage 9")
}

// =========================================================================
// error mapping
// =========================================================================

func (h *SessionHandler) writeSessionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, session.ErrNotFound):
		httperr.Write(w, http.StatusNotFound, "session_not_found", "session not found")
	case errors.Is(err, session.ErrInvalidTimeRange):
		httperr.Write(w, http.StatusBadRequest, "invalid_time_range", err.Error())
	case errors.Is(err, session.ErrInvalidQRTTL):
		httperr.Write(w, http.StatusBadRequest, "invalid_qr_ttl", err.Error())
	case errors.Is(err, session.ErrGroupsEmpty):
		httperr.Write(w, http.StatusBadRequest, "groups_empty", err.Error())
	case errors.Is(err, session.ErrGroupsNotInCourse):
		httperr.Write(w, http.StatusConflict, "groups_not_in_course_streams", err.Error())
	case errors.Is(err, session.ErrInvalidStatusTransition):
		httperr.Write(w, http.StatusConflict, "invalid_status_transition", err.Error())
	case errors.Is(err, policy.ErrNotFound), errors.Is(err, policy.ErrNoDefault):
		httperr.Write(w, http.StatusBadRequest, "policy_not_found", err.Error())
	case errors.Is(err, catalog.ErrCourseNotFound),
		errors.Is(err, catalog.ErrClassroomNotFound),
		errors.Is(err, catalog.ErrGroupNotFound):
		httperr.Write(w, http.StatusBadRequest, "catalog_not_found", err.Error())
	case errors.Is(err, errNotAuthorized):
		httperr.Write(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		// Специальное "not authorized" возвращается через строковое сравнение —
		// service уже возвращает стандартную ошибку. Проверим сигнатуру.
		if err != nil && err.Error() == "session: not authorized" {
			httperr.Write(w, http.StatusForbidden, "forbidden", "not authorized")
			return
		}
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

var errNotAuthorized = errors.New("session: not authorized")
