// Package qr — QR-ротатор: per-session goroutine, которая инкрементирует
// sessions.qr_counter и публикует подписанный токен в Hub.
//
// Manager реализует domain.RotatorController: session-service вызывает
// Start/Stop при переходах draft→active и active→closed. На старте API
// Bootstrap поднимает ротатор для всех status=active.
package qr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"attendance/internal/application/hub"
	"attendance/internal/domain"
	"attendance/internal/domain/session"
)

// QRBroadcast — JSON-сообщение, которое публикуется в Hub при каждой ротации.
// Teacher-UI ждёт `type=qr_token`, рендерит QR из `token`, ставит expiresAt-timer.
type QRBroadcast struct {
	Type      string    `json:"type"` // всегда "qr_token"
	SessionID string    `json:"session_id"`
	Counter   int       `json:"counter"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Deps — зависимости Manager.
type Deps struct {
	Sessions session.Repository
	Codec    domain.QRTokenCodec
	Hub      *hub.Hub
	Clock    domain.Clock
	Log      *slog.Logger
}

// Manager — единственная точка управления ротаторами. Thread-safe.
type Manager struct {
	Deps
	mu      sync.Mutex
	running map[uuid.UUID]*rotator
}

func NewManager(d Deps) *Manager {
	return &Manager{Deps: d, running: make(map[uuid.UUID]*rotator)}
}

var _ domain.RotatorController = (*Manager)(nil)

// Start поднимает ротатор для сессии. Идемпотентно: если уже запущен — no-op.
func (m *Manager) Start(sessionID uuid.UUID, secret []byte, ttlSeconds int) error {
	if len(secret) == 0 {
		return errors.New("rotator: empty secret")
	}
	if ttlSeconds < session.MinQRTTLSeconds || ttlSeconds > session.MaxQRTTLSeconds {
		return fmt.Errorf("rotator: ttl_seconds out of range: %d", ttlSeconds)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.running[sessionID]; ok {
		return nil // уже запущен
	}

	secretCopy := make([]byte, len(secret))
	copy(secretCopy, secret)

	ctx, cancel := context.WithCancel(context.Background())
	r := &rotator{
		sessionID: sessionID,
		secret:    secretCopy,
		ttl:       time.Duration(ttlSeconds) * time.Second,
		cancel:    cancel,
		deps:      m.Deps,
	}
	m.running[sessionID] = r

	go r.run(ctx)
	m.Log.Info("rotator: started",
		slog.String("session_id", sessionID.String()),
		slog.Int("ttl_seconds", ttlSeconds),
	)
	return nil
}

// Stop останавливает ротатор, закрывает WS-клиентов этой сессии.
// Идемпотентно: повторный вызов — no-op.
func (m *Manager) Stop(sessionID uuid.UUID) error {
	m.mu.Lock()
	r, ok := m.running[sessionID]
	if ok {
		delete(m.running, sessionID)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}
	r.cancel()
	m.Hub.CloseSession(sessionID)
	m.Log.Info("rotator: stopped", slog.String("session_id", sessionID.String()))
	return nil
}

// Shutdown останавливает все ротаторы для graceful shutdown.
func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.Lock()
	ids := make([]uuid.UUID, 0, len(m.running))
	for id, r := range m.running {
		r.cancel()
		ids = append(ids, id)
	}
	m.running = make(map[uuid.UUID]*rotator)
	m.mu.Unlock()
	for _, id := range ids {
		m.Hub.CloseSession(id)
	}
	m.Log.Info("rotator: shutdown complete", slog.Int("stopped", len(ids)))
}

// Bootstrap — при старте API находит все status=active сессии и поднимает
// для каждой ротатор. Восстанавливает ротацию после рестарта инстанса.
func (m *Manager) Bootstrap(ctx context.Context) error {
	sessions, err := m.Sessions.ActiveForBootstrap(ctx)
	if err != nil {
		return fmt.Errorf("rotator bootstrap: load active: %w", err)
	}
	for _, s := range sessions {
		if err := m.Start(s.ID, s.QRSecret, s.QRTTLSeconds); err != nil {
			m.Log.Warn("rotator: bootstrap start failed",
				slog.String("session_id", s.ID.String()),
				slog.String("err", err.Error()),
			)
		}
	}
	m.Log.Info("rotator: bootstrap", slog.Int("resumed", len(sessions)))
	return nil
}

// Subscribers — health/debug helper: сколько WS-клиентов подписано на session.
func (m *Manager) Subscribers(sessionID uuid.UUID) int {
	return m.Hub.Subscribers(sessionID)
}

// =========================================================================
// per-session goroutine
// =========================================================================

type rotator struct {
	sessionID uuid.UUID
	secret    []byte
	ttl       time.Duration
	cancel    context.CancelFunc
	deps      Deps
}

// run — основной цикл: сразу публикует первый QR (чтобы UI не ждал ttl),
// затем по тикеру каждые ttl секунд.
func (r *rotator) run(ctx context.Context) {
	// Первый тик — сразу.
	if err := r.tick(ctx); err != nil {
		r.deps.Log.Warn("rotator: initial tick failed",
			slog.String("session_id", r.sessionID.String()),
			slog.String("err", err.Error()),
		)
	}

	ticker := time.NewTicker(r.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.tick(ctx); err != nil {
				r.deps.Log.Warn("rotator: tick failed",
					slog.String("session_id", r.sessionID.String()),
					slog.String("err", err.Error()),
				)
			}
		}
	}
}

// tick = инкрементируем counter → собираем токен → публикуем.
func (r *rotator) tick(ctx context.Context) error {
	counter, err := r.deps.Sessions.IncrementQRCounter(ctx, r.sessionID)
	if err != nil {
		// Сессия исчезла (удалена/закрыта не через наш Stop) — самоостанавливаемся.
		if errors.Is(err, session.ErrNotFound) {
			r.deps.Log.Info("rotator: session gone, self-stopping",
				slog.String("session_id", r.sessionID.String()))
			r.cancel()
			return nil
		}
		return fmt.Errorf("increment counter: %w", err)
	}
	now := r.deps.Clock.Now(ctx)
	token, err := r.deps.Codec.Encode(r.secret, domain.QRToken{
		SessionID: r.sessionID,
		Counter:   counter,
		IssuedAt:  now,
	})
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	msg := QRBroadcast{
		Type:      "qr_token",
		SessionID: r.sessionID.String(),
		Counter:   counter,
		Token:     token,
		ExpiresAt: now.Add(r.ttl),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	r.deps.Hub.Broadcast(ctx, r.sessionID, data)
	return nil
}
