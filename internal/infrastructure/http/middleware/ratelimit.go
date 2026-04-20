package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"attendance/internal/infrastructure/http/httperr"
)

// IPRateLimiter — per-IP fixed-window лимитер.
//
// Используется на /auth/login, чтобы нельзя было брутфорсить пароль.
// 10 запросов в минуту с одного IP — разумный компромисс: человек не
// упирается, автоматика умирает.
//
// Реализация — списки timestamp'ов на IP. Чистим периодически, чтобы мапа
// не росла бесконечно. Для dev/MVP этого достаточно; при горизонтальном
// масштабировании нужно выносить в Redis (sliding-window через SET + SCRIPT).
type IPRateLimiter struct {
	window time.Duration
	limit  int

	mu      sync.Mutex
	buckets map[string][]time.Time

	stop chan struct{}
}

// NewIPRateLimiter стартует cleanup-goroutine и возвращает готовый к use
// middleware. Cleanup отрабатывает раз в window и удаляет истёкшие записи.
func NewIPRateLimiter(window time.Duration, limit int) *IPRateLimiter {
	l := &IPRateLimiter{
		window:  window,
		limit:   limit,
		buckets: make(map[string][]time.Time),
		stop:    make(chan struct{}),
	}
	go l.cleanup()
	return l
}

// Stop останавливает cleanup-goroutine. Для graceful shutdown API.
func (l *IPRateLimiter) Stop() {
	select {
	case <-l.stop:
		// уже закрыт
	default:
		close(l.stop)
	}
}

// Handle возвращает middleware. При превышении лимита — 429 +
// Retry-After header (рекомендация RFC 6585).
func (l *IPRateLimiter) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		l.mu.Lock()
		now := time.Now()
		times := filterRecent(l.buckets[ip], now, l.window)
		if len(times) >= l.limit {
			oldest := times[0]
			retryAfter := int(l.window.Seconds()-now.Sub(oldest).Seconds()) + 1
			if retryAfter < 1 {
				retryAfter = 1
			}
			l.buckets[ip] = times
			l.mu.Unlock()

			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			httperr.Write(w, http.StatusTooManyRequests, "rate_limited", "too many requests, retry later")
			return
		}
		times = append(times, now)
		l.buckets[ip] = times
		l.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (l *IPRateLimiter) cleanup() {
	t := time.NewTicker(l.window)
	defer t.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
			l.mu.Lock()
			now := time.Now()
			for ip, times := range l.buckets {
				filtered := filterRecent(times, now, l.window)
				if len(filtered) == 0 {
					delete(l.buckets, ip)
				} else {
					l.buckets[ip] = filtered
				}
			}
			l.mu.Unlock()
		}
	}
}

// filterRecent возвращает только те timestamp'ы, что попадают в окно.
// Работает in-place над копией среза — вызывающий получает новый slice,
// исходный (в мапе) не изменяется до assign'а.
func filterRecent(times []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	out := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
