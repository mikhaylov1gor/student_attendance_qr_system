// Package middleware содержит HTTP-middleware: auth, requirerole, slog-logger, recover.
package middleware

import (
	"net/http"
	"strings"

	"attendance/internal/domain/auth"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/http/httperr"
	"attendance/internal/platform/authctx"
)

// Auth возвращает middleware, которое:
//  1. Требует заголовок `Authorization: Bearer <jwt>`.
//  2. Проверяет подпись и exp JWT через AccessTokenSigner.
//  3. Кладёт Principal в context.
//
// Если токена нет или он невалиден — отдаёт 401 с унифицированным телом.
func Auth(signer auth.AccessTokenSigner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := extractBearer(r.Header.Get("Authorization"))
			if raw == "" {
				httperr.Write(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			claims, err := signer.Verify(raw)
			if err != nil {
				code := "invalid_token"
				if err == auth.ErrTokenExpired {
					code = "token_expired"
				}
				httperr.Write(w, http.StatusUnauthorized, code, err.Error())
				return
			}
			ctx := authctx.With(r.Context(), auth.Principal{
				UserID: claims.Subject,
				Role:   claims.Role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole — middleware, пропускающий только пользователей с одной из
// перечисленных ролей. Использовать ВСЕГДА после Auth.
func RequireRole(roles ...user.Role) func(http.Handler) http.Handler {
	allowed := make(map[user.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := authctx.From(r.Context())
			if !ok {
				httperr.Write(w, http.StatusUnauthorized, "unauthorized", "no principal in context")
				return
			}
			if _, pass := allowed[p.Role]; !pass {
				httperr.Write(w, http.StatusForbidden, "forbidden", "role not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractBearer(h string) string {
	const prefix = "Bearer "
	if len(h) < len(prefix) {
		return ""
	}
	if !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
