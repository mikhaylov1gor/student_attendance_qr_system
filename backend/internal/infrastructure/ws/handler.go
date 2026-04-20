package ws

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"attendance/internal/application/hub"
	"attendance/internal/domain/auth"
	"attendance/internal/domain/session"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/http/httperr"
)

// Deps — зависимости teacher-хендлера.
type Deps struct {
	Hub      *hub.Hub
	Signer   auth.AccessTokenSigner
	Sessions session.Repository
	Log      *slog.Logger
}

type TeacherHandler struct{ Deps }

func NewTeacherHandler(d Deps) *TeacherHandler { return &TeacherHandler{Deps: d} }

// bearerSubprotocol — префикс subprotocol-стринга, в которой клиент передаёт JWT.
// Пример: `Sec-WebSocket-Protocol: bearer.eyJhbGciOi...`. Мы отвечаем тем же
// subprotocol — иначе браузер закроет соединение.
const bearerSubprotocol = "bearer."

// TrackedConns — для graceful shutdown: закрываем все открытые соединения.
// WS-клиенты регистрируются в Hub, но Shutdown бьёт по списку всех clients.
var (
	trackedMu    sync.Mutex
	trackedConns = map[*Client]struct{}{}
)

// Serve — обработчик `/ws/sessions/:id/teacher`.
func (h *TeacherHandler) Serve(w http.ResponseWriter, r *http.Request) {
	// 1. Парсим session_id.
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_id", "not a uuid")
		return
	}

	// 2. Достаём JWT из Sec-WebSocket-Protocol.
	requestedProtos := websocket.AcceptOptions{
		Subprotocols: []string{},
	}
	jwtToken, selectedProto := extractBearerProtocol(r.Header.Values("Sec-WebSocket-Protocol"))
	if jwtToken == "" {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "missing bearer subprotocol")
		return
	}
	requestedProtos.Subprotocols = []string{selectedProto}

	// 3. Верификация JWT.
	claims, err := h.Signer.Verify(jwtToken)
	if err != nil {
		code := "invalid_token"
		if errors.Is(err, auth.ErrTokenExpired) {
			code = "token_expired"
		}
		httperr.Write(w, http.StatusUnauthorized, code, err.Error())
		return
	}
	if claims.Role != user.RoleTeacher && claims.Role != user.RoleAdmin {
		httperr.Write(w, http.StatusForbidden, "forbidden", "teacher/admin required")
		return
	}

	// 4. Проверяем сессию: существует, active, принадлежит teacher'у (или admin).
	sess, err := h.Sessions.GetByID(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			httperr.Write(w, http.StatusNotFound, "session_not_found", "session not found")
			return
		}
		h.Log.Error("ws: load session", slog.String("err", err.Error()))
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if sess.Status != session.StatusActive {
		httperr.Write(w, http.StatusConflict, "session_not_active", "session must be active")
		return
	}
	if claims.Role == user.RoleTeacher && sess.TeacherID != claims.Subject {
		httperr.Write(w, http.StatusForbidden, "forbidden", "not this session's teacher")
		return
	}

	// 5. Upgrade.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       requestedProtos.Subprotocols,
		InsecureSkipVerify: true, // dev: принимаем любой Origin. Для prod — ограничивать.
	})
	if err != nil {
		h.Log.Warn("ws: accept failed", slog.String("err", err.Error()))
		return
	}

	client := newClient(conn, sessionID, claims.Subject, h.Log)

	trackedMu.Lock()
	trackedConns[client] = struct{}{}
	trackedMu.Unlock()

	h.Hub.Register(client)
	h.Log.Info("ws: teacher connected",
		slog.String("session_id", sessionID.String()),
		slog.String("teacher_id", claims.Subject.String()),
	)

	// 6. Read/Write pumps в одной goroutine: write в main, read — в отдельной.
	// При любом выходе — снимаемся с Hub и списка.
	ctx := r.Context()
	go client.readPump(ctx)
	client.writePump(ctx)

	// cleanup
	_ = h.Hub.Unregister(client)
	trackedMu.Lock()
	delete(trackedConns, client)
	trackedMu.Unlock()
	h.Log.Info("ws: teacher disconnected", slog.String("session_id", sessionID.String()))
}

// Shutdown закрывает все активные соединения. Вызывается из cmd/api.
func Shutdown() {
	trackedMu.Lock()
	cs := make([]*Client, 0, len(trackedConns))
	for c := range trackedConns {
		cs = append(cs, c)
	}
	trackedConns = map[*Client]struct{}{}
	trackedMu.Unlock()
	for _, c := range cs {
		c.Close()
	}
}

// extractBearerProtocol ищет subprotocol вида `bearer.<jwt>`.
// Возвращает JWT и полный subprotocol-стринг для echo в upgrade-ответе.
func extractBearerProtocol(headers []string) (jwt, proto string) {
	// Заголовок может быть передан как "bearer.xxx, other.yyy"
	// (HTTP-формат comma-separated) или несколькими значениями.
	for _, h := range headers {
		for _, item := range strings.Split(h, ",") {
			item = strings.TrimSpace(item)
			if strings.HasPrefix(item, bearerSubprotocol) {
				return item[len(bearerSubprotocol):], item
			}
		}
	}
	return "", ""
}
