// Package clock содержит реализацию порта domain.Clock.
// RealClock возвращает текущее UTC-время. В тестах используется FixedClock.
package clock

import (
	"context"
	"time"
)

type RealClock struct{}

func New() RealClock { return RealClock{} }

func (RealClock) Now(_ context.Context) time.Time { return time.Now().UTC() }

// FixedClock полезен в тестах и при ручных сценариях (seed, backfill).
type FixedClock struct{ T time.Time }

func Fixed(t time.Time) FixedClock { return FixedClock{T: t} }

func (f FixedClock) Now(_ context.Context) time.Time { return f.T }
