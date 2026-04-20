package checks_test

import (
	"context"
	"testing"

	"attendance/internal/domain/policy"
	"attendance/internal/domain/policy/checks"
)

func strPtr(s string) *string { return &s }

func TestWiFiCheck(t *testing.T) {
	t.Parallel()

	c := checks.NewWiFiCheck()

	cfg := func(enabled, fromClassroom bool, extras ...string) policy.MechanismsConfig {
		return policy.MechanismsConfig{
			WiFi: policy.WiFiConfig{
				Enabled:                     enabled,
				RequiredBSSIDsFromClassroom: fromClassroom,
				ExtraBSSIDs:                 extras,
			},
		}
	}
	in := func(clientBssid *string, allowed ...string) policy.CheckInput {
		return policy.CheckInput{
			AllowedBSSIDs: allowed,
			ClientBSSID:   clientBssid,
		}
	}

	t.Run("disabled → skipped", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(false, true), in(strPtr("aa:bb:cc:dd:ee:01"), "aa:bb:cc:dd:ee:01"))
		if res.Status != policy.StatusSkipped || res.Details["reason"] != policy.ReasonDisabled {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("no client data → skipped", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(true, true), in(nil, "aa:bb:cc:dd:ee:01"))
		if res.Status != policy.StatusSkipped || res.Details["reason"] != policy.ReasonNoClientData {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("empty client bssid → skipped", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(true, true), in(strPtr(""), "aa:bb:cc:dd:ee:01"))
		if res.Status != policy.StatusSkipped || res.Details["reason"] != policy.ReasonNoClientData {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("enabled но allowlist пустой → skipped (no_allowed)", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(true, false), in(strPtr("aa:bb:cc:dd:ee:01")))
		if res.Status != policy.StatusSkipped || res.Details["reason"] != policy.ReasonNoAllowed {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("совпадение из classroom → passed", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(true, true), in(strPtr("aa:bb:cc:dd:ee:01"), "aa:bb:cc:dd:ee:01", "aa:bb:cc:dd:ee:02"))
		if res.Status != policy.StatusPassed {
			t.Fatalf("status = %s, want passed; details=%v", res.Status, res.Details)
		}
	})

	t.Run("совпадение из extras → passed", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(true, false, "AA:BB:CC:DD:EE:99"), in(strPtr("aa:bb:cc:dd:ee:99")))
		if res.Status != policy.StatusPassed {
			t.Fatalf("status = %s, want passed", res.Status)
		}
	})

	t.Run("нет совпадения → failed", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfg(true, true), in(strPtr("11:22:33:44:55:66"), "aa:bb:cc:dd:ee:01"))
		if res.Status != policy.StatusFailed {
			t.Fatalf("status = %s, want failed", res.Status)
		}
	})

	t.Run("нормализация BSSID (регистр / дефисы)", func(t *testing.T) {
		t.Parallel()
		// allowlist: "AA-BB-CC-DD-EE-01", клиент: "aa:bb:cc:dd:ee:01"
		res, _ := c.Check(context.Background(), cfg(true, true), in(strPtr("aa:bb:cc:dd:ee:01"), "AA-BB-CC-DD-EE-01"))
		if res.Status != policy.StatusPassed {
			t.Fatalf("status = %s, want passed (normalization)", res.Status)
		}
	})
}
