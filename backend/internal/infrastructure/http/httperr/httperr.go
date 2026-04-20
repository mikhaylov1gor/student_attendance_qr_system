// Package httperr — формат ошибок API и helper'ы для хендлеров.
//
// Все ошибки наружу летят в виде:
//
//	{ "error": { "code": "invalid_credentials", "message": "..." } }
//
// Коды стабильны (машинно-читаемые), message — для человека.
package httperr

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Body — формат тела ответа при ошибке.
type Body struct {
	Error Error `json:"error"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Write сериализует ошибку в ResponseWriter.
func Write(w http.ResponseWriter, status int, code, message string) {
	WriteDetails(w, status, code, message, nil)
}

func WriteDetails(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Body{Error: Error{Code: code, Message: message, Details: details}})
}

// WriteJSON — успешный ответ, произвольный payload.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// LogUnexpected — логируем 5xx-ошибки в stdout (чтобы не зашумлять reply клиенту).
func LogUnexpected(log *slog.Logger, r *http.Request, err error) {
	if err == nil {
		return
	}
	log.ErrorContext(r.Context(), "handler failed",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.String("err", err.Error()),
	)
}

// As — мини-обёртка вокруг errors.As для более короткого кода в хендлерах.
func As[T error](err error, out *T) bool { return errors.As(err, out) }
