// Package http — сборка chi-роутера и регистрация хендлеров.
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"

	"attendance/internal/domain/auth"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/http/handlers"
	appmid "attendance/internal/infrastructure/http/middleware"
)

// Deps — всё, что нужно роутеру для регистрации маршрутов.
type Deps struct {
	Log          *slog.Logger
	Signer       auth.AccessTokenSigner
	AuthH        *handlers.AuthHandler
	PolicyH      *handlers.PolicyHandler
	AuditH       *handlers.AuditHandler
	CatalogH     *handlers.CatalogHandler
	SessionH     *handlers.SessionHandler
	AttendanceH  *handlers.AttendanceHandler
	UserH        *handlers.UserHandler
	StudentMeH   *handlers.StudentMeHandler
	ReportH      *handlers.ReportHandler
	WSHandler    http.HandlerFunc // teacher-WS хендлер
	Health       http.HandlerFunc // /healthz handler
	CORSOrigins  []string         // nil/empty → CORS выключен
	LoginLimiter *appmid.IPRateLimiter
}

// NewRouter собирает chi-роутер: middleware, public-роуты, protected-роуты.
func NewRouter(d Deps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimid.RequestID)
	r.Use(chimid.RealIP)
	r.Use(chimid.Recoverer)
	r.Use(appmid.RequestMeta())
	r.Use(appmid.SlogLogger(d.Log))

	// CORS — один глобальный middleware до всех роутов, включая WS.
	if len(d.CORSOrigins) > 0 {
		r.Use(appmid.CORS(d.CORSOrigins))
	}

	// Public
	r.Get("/healthz", d.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			// Login под rate-limit'ом по IP (brute-force защита).
			if d.LoginLimiter != nil {
				r.With(d.LoginLimiter.Handle).Post("/login", d.AuthH.Login)
			} else {
				r.Post("/login", d.AuthH.Login)
			}
			r.Post("/refresh", d.AuthH.Refresh)

			// Logout/Me требуют access-токен.
			r.Group(func(pr chi.Router) {
				pr.Use(appmid.Auth(d.Signer))
				pr.Post("/logout", d.AuthH.Logout)
				pr.Get("/me", d.AuthH.Me)
			})
		})

		// Admin-only CRUD политик безопасности.
		r.Route("/policies", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleAdmin))
			r.Get("/", d.PolicyH.List)
			r.Post("/", d.PolicyH.Create)
			r.Get("/{id}", d.PolicyH.Get)
			r.Patch("/{id}", d.PolicyH.Update)
			r.Delete("/{id}", d.PolicyH.Delete)
			r.Post("/{id}/set-default", d.PolicyH.SetDefault)
		})

		// Admin-only audit log.
		r.Route("/audit", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleAdmin))
			r.Get("/", d.AuditH.List)
			r.Post("/verify", d.AuditH.Verify)
		})

		// Catalog — GET доступен всем аутентифицированным;
		//           мутации — только admin.
		catalogReadRoles := []user.Role{user.RoleAdmin, user.RoleTeacher, user.RoleStudent}
		{
			r.Route("/courses", func(r chi.Router) {
				r.Use(appmid.Auth(d.Signer))
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(catalogReadRoles...))
					pr.Get("/", d.CatalogH.ListCourses)
					pr.Get("/{id}", d.CatalogH.GetCourse)
				})
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(user.RoleAdmin))
					pr.Post("/", d.CatalogH.CreateCourse)
					pr.Patch("/{id}", d.CatalogH.UpdateCourse)
					pr.Delete("/{id}", d.CatalogH.DeleteCourse)
				})
			})

			r.Route("/groups", func(r chi.Router) {
				r.Use(appmid.Auth(d.Signer))
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(catalogReadRoles...))
					pr.Get("/", d.CatalogH.ListGroups)
					pr.Get("/{id}", d.CatalogH.GetGroup)
				})
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(user.RoleAdmin))
					pr.Post("/", d.CatalogH.CreateGroup)
					pr.Patch("/{id}", d.CatalogH.UpdateGroup)
					pr.Delete("/{id}", d.CatalogH.DeleteGroup)
				})
			})

			r.Route("/streams", func(r chi.Router) {
				r.Use(appmid.Auth(d.Signer))
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(catalogReadRoles...))
					pr.Get("/", d.CatalogH.ListStreams) // ?course_id=...
					pr.Get("/{id}", d.CatalogH.GetStream)
				})
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(user.RoleAdmin))
					pr.Post("/", d.CatalogH.CreateStream)
					pr.Patch("/{id}", d.CatalogH.UpdateStream)
					pr.Delete("/{id}", d.CatalogH.DeleteStream)
				})
			})

			r.Route("/classrooms", func(r chi.Router) {
				r.Use(appmid.Auth(d.Signer))
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(catalogReadRoles...))
					pr.Get("/", d.CatalogH.ListClassrooms)
					pr.Get("/{id}", d.CatalogH.GetClassroom)
				})
				r.Group(func(pr chi.Router) {
					pr.Use(appmid.RequireRole(user.RoleAdmin))
					pr.Post("/", d.CatalogH.CreateClassroom)
					pr.Patch("/{id}", d.CatalogH.UpdateClassroom)
					pr.Delete("/{id}", d.CatalogH.DeleteClassroom)
				})
			})
		}

		// Sessions — создание и lifecycle доступны teacher+admin.
		r.Route("/sessions", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleTeacher, user.RoleAdmin))
			r.Post("/", d.SessionH.Create)
			r.Get("/", d.SessionH.List)
			r.Get("/{id}", d.SessionH.Get)
			r.Patch("/{id}", d.SessionH.Update)
			r.Delete("/{id}", d.SessionH.Delete)
			r.Post("/{id}/start", d.SessionH.Start)
			r.Post("/{id}/close", d.SessionH.Close)
			r.Get("/{id}/attendance", d.SessionH.Attendance) // заготовка
		})

		// Attendance — только студент (преподаватель смотрит через WS + /sessions/:id/attendance).
		r.Route("/attendance", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleStudent))
			r.Post("/", d.AttendanceH.Submit)
		})

		// Admin-only CRUD пользователей.
		r.Route("/users", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleAdmin))
			r.Get("/", d.UserH.List)
			r.Post("/", d.UserH.Create)
			r.Get("/{id}", d.UserH.Get)
			r.Patch("/{id}", d.UserH.Update)
			r.Delete("/{id}", d.UserH.Delete)
			r.Post("/{id}/reset-password", d.UserH.ResetPassword)
		})

		// Self-service для студента.
		r.Route("/students/me", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleStudent))
			r.Get("/attendance", d.StudentMeH.Attendance)
			r.Get("/stats", d.StudentMeH.Stats)
		})

		// Отчёты — teacher + admin. Teacher автоматически ограничен своими
		// сессиями на уровне handler'а.
		r.Route("/reports", func(r chi.Router) {
			r.Use(appmid.Auth(d.Signer))
			r.Use(appmid.RequireRole(user.RoleTeacher, user.RoleAdmin))
			r.Get("/attendance.{format}", d.ReportH.Attendance)
		})
	})

	// WebSocket teacher-канал. JWT идёт через Sec-WebSocket-Protocol,
	// поэтому не подключаем Auth-middleware — он требует Authorization-header.
	if d.WSHandler != nil {
		r.Get("/ws/sessions/{id}/teacher", d.WSHandler)
	}

	// 404 / 405 в унифицированном формате.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"route not found"}}`))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":{"code":"method_not_allowed","message":"method not allowed"}}`))
	})

	return r
}
