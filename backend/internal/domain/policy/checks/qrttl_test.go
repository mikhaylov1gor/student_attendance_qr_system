package checks_test

import (
	"context"
	"testing"

	"attendance/internal/domain/policy"
	"attendance/internal/domain/policy/checks"
)

func TestQRTTLCheck(t *testing.T) {
	t.Parallel()

	c := checks.NewQRTTLCheck()

	if c.Name() != policy.MechanismQRTTL {
		t.Fatalf("Name = %s, want %s", c.Name(), policy.MechanismQRTTL)
	}

	tests := []struct {
		name           string
		enabled        bool
		tokenCounter   int
		currentCounter int
		wantStatus     policy.CheckStatus
		wantReason     string // "" = без проверки
	}{
		{"disabled → skipped", false, 0, 0, policy.StatusSkipped, policy.ReasonDisabled},
		{"counter == current → passed", true, 10, 10, policy.StatusPassed, ""},
		{"counter = current-1 → passed (slack)", true, 9, 10, policy.StatusPassed, ""},
		{"counter = current-2 → passed (boundary)", true, 8, 10, policy.StatusPassed, ""},
		{"counter = current-3 → failed (stale)", true, 7, 10, policy.StatusFailed, policy.ReasonStale},
		{"counter > current → failed (future)", true, 11, 10, policy.StatusFailed, policy.ReasonFutureCounter},
		{"counter far in past → failed (stale)", true, 0, 10, policy.StatusFailed, policy.ReasonStale},
	}

	cfgFor := func(en bool) policy.MechanismsConfig {
		return policy.MechanismsConfig{
			QRTTL: policy.QRTTLConfig{Enabled: en, TTLSeconds: 10},
		}
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := c.Check(context.Background(), cfgFor(tt.enabled), policy.CheckInput{
				TokenCounter:   tt.tokenCounter,
				CurrentCounter: tt.currentCounter,
			})
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if res.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s", res.Status, tt.wantStatus)
			}
			if tt.wantReason != "" {
				got, _ := res.Details["reason"].(string)
				if got != tt.wantReason {
					t.Errorf("reason = %q, want %q", got, tt.wantReason)
				}
			}
		})
	}
}

func TestQRTTLCheck_SlackConstant(t *testing.T) {
	// Явная проверка, что slack = 2, а не случайно изменилась.
	if checks.QRTTLCounterSlack != 2 {
		t.Fatalf("QRTTLCounterSlack = %d, want 2", checks.QRTTLCounterSlack)
	}
}
