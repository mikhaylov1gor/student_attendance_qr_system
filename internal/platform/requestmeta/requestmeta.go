// Package requestmeta — request-scoped метаданные, которые попадают из HTTP
// в сервисный слой через context. Используется audit-сервисом: IP и UserAgent
// не хочется прокидывать параметром через 5 функций.
package requestmeta

import (
	"context"
	"net"
)

type ctxKey int

const metaKey ctxKey = 0

// Meta — что middleware кладёт в ctx.
type Meta struct {
	RemoteIP  net.IP
	UserAgent string
}

func With(ctx context.Context, m Meta) context.Context {
	return context.WithValue(ctx, metaKey, m)
}

func From(ctx context.Context) Meta {
	m, _ := ctx.Value(metaKey).(Meta)
	return m
}
