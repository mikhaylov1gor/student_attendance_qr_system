package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chi "github.com/go-chi/chi/v5/middleware"
)

// SlogLogger пишет по строчке slog на каждый завершённый запрос.
// RequestID берётся из chi/middleware (нужно подключить до этого middleware).
func SlogLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chi.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.InfoContext(r.Context(), "http",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("elapsed", time.Since(start)),
				slog.String("request_id", chi.GetReqID(r.Context())),
				slog.String("remote", r.RemoteAddr),
			)
		})
	}
}
