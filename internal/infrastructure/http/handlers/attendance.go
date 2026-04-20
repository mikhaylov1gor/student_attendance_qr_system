package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	appattendance "attendance/internal/application/attendance"
	"attendance/internal/domain/attendance"
	"attendance/internal/domain/session"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
)

type AttendanceHandler struct {
	svc *appattendance.Service
	log *slog.Logger
}

func NewAttendanceHandler(svc *appattendance.Service, log *slog.Logger) *AttendanceHandler {
	return &AttendanceHandler{svc: svc, log: log}
}

// Submit — POST /api/v1/attendance. Принципал — студент (через JWT).
func (h *AttendanceHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req dto.SubmitAttendanceRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	clientTime := time.Time{}
	if req.ClientTime != nil {
		clientTime = *req.ClientTime
	}
	res, err := h.svc.Submit(r.Context(), appattendance.SubmitInput{
		QRToken:    req.QRToken,
		GeoLat:     req.GeoLat,
		GeoLng:     req.GeoLng,
		BSSID:      req.BSSID,
		ClientTime: clientTime,
	})
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.AttendanceFromDomain(res.Record, res.Checks))
}

func (h *AttendanceHandler) writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, attendance.ErrInvalidQRToken):
		httperr.Write(w, http.StatusBadRequest, "invalid_qr_token", "qr token invalid or malformed")
	case errors.Is(err, attendance.ErrAlreadySubmitted):
		httperr.Write(w, http.StatusConflict, "already_submitted", "attendance already submitted for this session")
	case errors.Is(err, session.ErrNotAcceptingAttendance):
		httperr.Write(w, http.StatusConflict, "session_not_accepting", "session is not active or out of time range")
	default:
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
