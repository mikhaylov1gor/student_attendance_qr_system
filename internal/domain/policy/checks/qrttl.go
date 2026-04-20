// Package checks содержит конкретные реализации policy.SecurityCheck.
// Регистрируются в composition root (cmd/api/main.go).
package checks

import (
	"context"

	"attendance/internal/domain/policy"
)

// QRTTLCounterSlack — насколько counter в токене может отставать от текущего
// counter сессии, чтобы считать токен свежим. slack=2 покрывает гонку
// «ротатор только что крутнул, студент жмёт подтверждение с токеном прошлого
// тика» + сетевую задержку.
const QRTTLCounterSlack = 2

// QRTTLCheck — проверка свежести QR-токена по счётчику сессии.
//
// Оперирует только счётчиками (TokenCounter / CurrentCounter). Wallclock
// (TokenIssuedAt) не используется: при расхождении серверного времени
// счётчики остаются достоверным источником порядка ротаций.
type QRTTLCheck struct{}

func NewQRTTLCheck() *QRTTLCheck { return &QRTTLCheck{} }

func (c *QRTTLCheck) Name() string { return policy.MechanismQRTTL }

func (c *QRTTLCheck) Check(
	_ context.Context,
	cfg policy.MechanismsConfig,
	input policy.CheckInput,
) (policy.CheckResult, error) {
	if !cfg.QRTTL.Enabled {
		return skippedResult(policy.MechanismQRTTL, policy.ReasonDisabled, nil), nil
	}

	tc := input.TokenCounter
	cc := input.CurrentCounter

	// Токен с counter'ом из будущего — подделка/рассинхронизация.
	if tc > cc {
		return policy.CheckResult{
			Mechanism: policy.MechanismQRTTL,
			Status:    policy.StatusFailed,
			Details: map[string]any{
				"reason":          policy.ReasonFutureCounter,
				"token_counter":   tc,
				"current_counter": cc,
			},
		}, nil
	}

	if cc-tc > QRTTLCounterSlack {
		return policy.CheckResult{
			Mechanism: policy.MechanismQRTTL,
			Status:    policy.StatusFailed,
			Details: map[string]any{
				"reason":          policy.ReasonStale,
				"token_counter":   tc,
				"current_counter": cc,
				"slack":           QRTTLCounterSlack,
			},
		}, nil
	}

	return policy.CheckResult{
		Mechanism: policy.MechanismQRTTL,
		Status:    policy.StatusPassed,
		Details: map[string]any{
			"token_counter":   tc,
			"current_counter": cc,
			"slack":           QRTTLCounterSlack,
		},
	}, nil
}

// skippedResult — унифицированный конструктор skipped-CheckResult для всех чеков.
func skippedResult(mech, reason string, extra map[string]any) policy.CheckResult {
	d := map[string]any{"reason": reason}
	for k, v := range extra {
		d[k] = v
	}
	return policy.CheckResult{Mechanism: mech, Status: policy.StatusSkipped, Details: d}
}
