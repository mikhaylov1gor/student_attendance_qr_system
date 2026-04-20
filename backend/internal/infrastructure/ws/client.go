// Package ws — WebSocket-хендлер для teacher-каналов и обёртка Client,
// реализующая hub.Client поверх coder/websocket.
package ws

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"attendance/internal/application/hub"
)

// Client — реализация hub.Client на coder/websocket.Conn.
//
// Поведение: под каждое соединение отдельная send-chan ёмкостью sendBuffer.
// Если клиент не читает быстрее, чем мы шлём — переполнение → закрываем
// соединение с close-code 1011 (internal). Это защитная мера: teacher-UI
// должен либо успевать, либо переподключаться.
type Client struct {
	conn      *websocket.Conn
	sessionID uuid.UUID
	teacherID uuid.UUID
	send      chan []byte
	log       *slog.Logger

	closeOnce sync.Once
	closed    chan struct{}
}

const (
	sendBuffer   = 32
	writeTimeout = 5 * time.Second
)

var _ hub.Client = (*Client)(nil)

// newClient создаётся хендлером после успешного upgrade.
func newClient(conn *websocket.Conn, sessionID, teacherID uuid.UUID, log *slog.Logger) *Client {
	return &Client{
		conn:      conn,
		sessionID: sessionID,
		teacherID: teacherID,
		send:      make(chan []byte, sendBuffer),
		closed:    make(chan struct{}),
		log:       log,
	}
}

func (c *Client) SessionID() uuid.UUID { return c.sessionID }

// Send — неблокирующая публикация. Если send-chan полон — считаем клиента
// отвалившимся и закрываем соединение.
func (c *Client) Send(_ context.Context, msg []byte) error {
	select {
	case <-c.closed:
		return errors.New("ws: client closed")
	case c.send <- msg:
		return nil
	default:
		c.log.Warn("ws: send buffer full, closing",
			slog.String("session_id", c.sessionID.String()))
		c.Close()
		return errors.New("ws: send buffer full")
	}
}

// Close — идемпотентное закрытие соединения с close-code 1001 (going away).
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.conn.Close(websocket.StatusGoingAway, "session closed")
	})
}

// writePump читает из send-chan и пишет в ws. Завершается при close(closed)
// или ошибке записи. Также периодически пингует соединение для поддержания.
func (c *Client) writePump(ctx context.Context) {
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closed:
			return
		case msg := <-c.send:
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(wctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				c.log.Debug("ws: write failed",
					slog.String("session_id", c.sessionID.String()),
					slog.String("err", err.Error()))
				c.Close()
				return
			}
		case <-ping.C:
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(wctx)
			cancel()
			if err != nil {
				c.Close()
				return
			}
		}
	}
}

// readPump тихо читает всё, что пришлёт клиент (мы не ожидаем сообщений от
// teacher'а по этому каналу — он только слушает QR). Закрывает соединение,
// если клиент нагенерит текст, или просто сигнализирует о его уходе.
func (c *Client) readPump(ctx context.Context) {
	defer c.Close()
	for {
		_, _, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
	}
}
