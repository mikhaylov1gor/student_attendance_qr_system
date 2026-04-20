package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appreport "attendance/internal/application/report"
	"attendance/internal/domain/report"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/http/httperr"
	infrareport "attendance/internal/infrastructure/report"
	"attendance/internal/platform/authctx"
)

type ReportHandler struct {
	svc *appreport.Service
	log *slog.Logger
}

func NewReportHandler(svc *appreport.Service, log *slog.Logger) *ReportHandler {
	return &ReportHandler{svc: svc, log: log}
}

// Attendance — GET /api/v1/reports/attendance.{xlsx,csv}
// Query: session_id | group_id | course_id (один обязателен), from, to (RFC3339).
func (h *ReportHandler) Attendance(w http.ResponseWriter, r *http.Request) {
	principal, err := authctx.Require(r.Context())
	if err != nil {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "no principal")
		return
	}

	// Формат — из URL-параметра chi ({format:xlsx|csv}).
	format := strings.ToLower(chi.URLParam(r, "format"))
	if format != "xlsx" && format != "csv" {
		httperr.Write(w, http.StatusBadRequest, "invalid_format", "format must be xlsx or csv")
		return
	}

	f, err := parseReportFilter(r)
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_filter", err.Error())
		return
	}

	// Teacher-scoping: teacher видит только свои сессии.
	if principal.Role == user.RoleTeacher {
		uid := principal.UserID
		f.TeacherID = &uid
	}

	rows, err := h.svc.GenerateAttendance(r.Context(), f)
	if err != nil {
		switch {
		case errors.Is(err, appreport.ErrNoFilter),
			errors.Is(err, appreport.ErrAmbiguousFilter):
			httperr.Write(w, http.StatusBadRequest, "invalid_filter", err.Error())
		default:
			httperr.LogUnexpected(h.log, r, err)
			httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		}
		return
	}

	fname := buildFilename(f, format)

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+fname+`"`)
		w.WriteHeader(http.StatusOK)
		if err := infrareport.WriteCSV(w, rows); err != nil {
			h.log.ErrorContext(r.Context(), "csv write", slog.String("err", err.Error()))
		}
	case "xlsx":
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="`+fname+`"`)
		w.WriteHeader(http.StatusOK)
		if err := infrareport.WriteXLSX(w, rows); err != nil {
			h.log.ErrorContext(r.Context(), "xlsx write", slog.String("err", err.Error()))
		}
	}
}

func parseReportFilter(r *http.Request) (report.Filter, error) {
	q := r.URL.Query()
	f := report.Filter{}
	optUUID := func(name string, dst **uuid.UUID) error {
		raw := q.Get(name)
		if raw == "" {
			return nil
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return fmt.Errorf("%s: not a uuid", name)
		}
		*dst = &id
		return nil
	}
	if err := optUUID("session_id", &f.SessionID); err != nil {
		return f, err
	}
	if err := optUUID("group_id", &f.GroupID); err != nil {
		return f, err
	}
	if err := optUUID("course_id", &f.CourseID); err != nil {
		return f, err
	}
	if raw := q.Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return f, fmt.Errorf("from: not RFC3339")
		}
		f.From = &t
	}
	if raw := q.Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return f, fmt.Errorf("to: not RFC3339")
		}
		f.To = &t
	}
	return f, nil
}

// buildFilename — человекочитаемое имя attachment-файла.
// Пример: attendance_session_656500fb.xlsx
func buildFilename(f report.Filter, format string) string {
	var key string
	switch {
	case f.SessionID != nil:
		key = "session_" + shortUUID(*f.SessionID)
	case f.GroupID != nil:
		key = "group_" + shortUUID(*f.GroupID)
	case f.CourseID != nil:
		key = "course_" + shortUUID(*f.CourseID)
	default:
		key = "all"
	}
	return fmt.Sprintf("attendance_%s.%s", key, format)
}

func shortUUID(id uuid.UUID) string { return id.String()[:8] }
