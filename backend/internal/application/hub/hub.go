// Package hub — in-memory pub/sub для WebSocket-каналов преподавателей.
//
// Rotator публикует QR-событие → Hub фан-аутит его всем клиентам, подписанным
// на этот session_id. Attendance-сервис после успешного Submit публикует
// attendance-событие тому же session_id.
//
// Реализация преднамеренно простая: один инстанс API, in-memory. Масштабирование
// на несколько инстансов — это уже pub/sub через Redis/NATS, выходит за MVP.
package hub

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Client — абстракция подписчика. WS-хендлер оборачивает websocket.Conn
// в свою реализацию Client и регистрирует в Hub.
type Client interface {
	SessionID() uuid.UUID
	// Send передаёт сообщение клиенту. Неблокирующая: если клиент отстал,
	// реализация вольна дропнуть сообщение и вернуть ошибку (Hub unregister'нет).
	Send(ctx context.Context, msg []byte) error
	// Close завершает соединение с WS-статусом going_away. Идемпотентна.
	Close()
}

// Hub хранит подписчиков по session_id. Thread-safe.
type Hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]map[Client]struct{}
	log   *slog.Logger
}

func New(log *slog.Logger) *Hub {
	return &Hub{
		rooms: make(map[uuid.UUID]map[Client]struct{}),
		log:   log,
	}
}

// ErrNotSubscribed возвращается Unregister'ом, если клиент не зарегистрирован.
var ErrNotSubscribed = errors.New("hub: client not subscribed")

// Register добавляет клиента в комнату session_id.
func (h *Hub) Register(c Client) {
	sid := c.SessionID()
	h.mu.Lock()
	room, ok := h.rooms[sid]
	if !ok {
		room = make(map[Client]struct{})
		h.rooms[sid] = room
	}
	room[c] = struct{}{}
	size := len(room)
	h.mu.Unlock()
	h.log.Debug("hub: register",
		slog.String("session_id", sid.String()),
		slog.Int("room_size", size),
	)
}

// Unregister убирает клиента из комнаты. Если комната опустела — удаляем её
// (чтобы мапа не росла при закрытии сессий).
func (h *Hub) Unregister(c Client) error {
	sid := c.SessionID()
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[sid]
	if !ok {
		return ErrNotSubscribed
	}
	if _, present := room[c]; !present {
		return ErrNotSubscribed
	}
	delete(room, c)
	if len(room) == 0 {
		delete(h.rooms, sid)
	}
	h.log.Debug("hub: unregister",
		slog.String("session_id", sid.String()),
		slog.Int("room_size", len(room)),
	)
	return nil
}

// Broadcast рассылает msg всем клиентам комнаты sessionID.
// Клиенты, чей Send вернул ошибку (например, backpressure), логируются —
// но не отключаются отсюда: их отвалит write-loop при следующей попытке.
// Реализация клиента (ws/client.go) должна делать Close + Unregister сама.
func (h *Hub) Broadcast(ctx context.Context, sessionID uuid.UUID, msg []byte) {
	h.mu.RLock()
	room := h.rooms[sessionID]
	// Копируем клиентов, чтобы Send не держал RLock.
	clients := make([]Client, 0, len(room))
	for c := range room {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		if err := c.Send(ctx, msg); err != nil {
			h.log.Debug("hub: send failed",
				slog.String("session_id", sessionID.String()),
				slog.String("err", err.Error()),
			)
		}
	}
}

// CloseSession отключает всех клиентов комнаты и удаляет комнату.
// Вызывается при SessionService.Close (и Rotator.Stop): как только ротация
// остановлена, держать teacher-WS бессмысленно.
func (h *Hub) CloseSession(sessionID uuid.UUID) {
	h.mu.Lock()
	room := h.rooms[sessionID]
	delete(h.rooms, sessionID)
	h.mu.Unlock()
	for c := range room {
		c.Close()
	}
	h.log.Debug("hub: close session",
		slog.String("session_id", sessionID.String()),
		slog.Int("clients_closed", len(room)),
	)
}

// Shutdown закрывает все комнаты. Для graceful shutdown API.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	rooms := h.rooms
	h.rooms = make(map[uuid.UUID]map[Client]struct{})
	h.mu.Unlock()
	for _, room := range rooms {
		for c := range room {
			c.Close()
		}
	}
	h.log.Info("hub: shutdown complete")
}

// Subscribers возвращает число подписчиков на sessionID. Для health-эндпоинтов.
func (h *Hub) Subscribers(sessionID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[sessionID])
}
