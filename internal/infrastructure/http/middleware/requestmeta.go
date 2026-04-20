package middleware

import (
	"net"
	"net/http"

	"attendance/internal/platform/requestmeta"
)

// RequestMeta кладёт в ctx IP клиента и User-Agent.
// RealIP-middleware chi выставляет r.RemoteAddr из X-Forwarded-For / X-Real-IP,
// так что берём RemoteAddr и вычленяем ip:port.
func RequestMeta() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			m := requestmeta.Meta{
				RemoteIP:  net.ParseIP(host),
				UserAgent: r.UserAgent(),
			}
			ctx := requestmeta.With(r.Context(), m)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
