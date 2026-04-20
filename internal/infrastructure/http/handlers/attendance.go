package handlers

import (
	"log/slog"
	"net/http"
	"time"

	appattendance "attendance/internal/application/attendance"
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
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.AttendanceFromDomain(res.Record, res.Checks))
}
