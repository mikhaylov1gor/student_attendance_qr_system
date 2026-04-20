package middleware

import (
	"net/http"
	"strings"
)

// CORS — простейшая реализация CORS с allowlist'ом Origins из env.
//
// Принципы:
//   - разрешаем запрос только если Origin в allowlist (иначе не добавляем
//     никаких CORS-заголовков → браузер заблокирует CORS-запрос);
//   - credentials=true: фронту нужно передавать Bearer-токен в Authorization;
//   - preflight (OPTIONS) возвращаем 204 без пропуска внутрь router'а;
//   - Vary: Origin, чтобы CDN-прокси не кэшировал неверный ответ.
//
// Список разрешённых заголовков заданы статически — для нашего API этого
// хватает (Authorization + Content-Type). WebSocket-протокол добавляется для
// случая, когда frontend захочет присоединяться к /ws/* через тот же домен.
var corsAllowedHeaders = strings.Join([]string{
	"Authorization",
	"Content-Type",
	"X-Request-ID",
	"Sec-WebSocket-Protocol",
}, ", ")

var corsAllowedMethods = strings.Join([]string{
	"GET", "POST", "PATCH", "DELETE", "OPTIONS",
}, ", ")

// CORS возвращает middleware. origins — список разрешённых Origin'ов
// (например, `http://localhost:5173`). Пустой список → CORS выключен.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Add("Vary", "Origin")
				}
			}

			// Preflight — отвечаем 204 без проброса в router.
			if r.Method == http.MethodOptions && origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
					w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
					w.Header().Set("Access-Control-Max-Age", "86400") // 24 часа
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
