package checks

import (
	"context"
	"strings"

	"attendance/internal/domain/policy"
)

// WiFiCheck — проверка привязки к Wi-Fi-аудитории (BSSID).
//
// Логика allowlist:
//   - если cfg.WiFi.RequiredBSSIDsFromClassroom=true → добавляем classroom.AllowedBSSIDs;
//   - ExtraBSSIDs из политики добавляются всегда;
//   - пустой allowlist → skipped (no_allowed).
//
// BSSID нормализуются: lowercase + trim + убираем разделители `-` и `:`,
// чтобы aa:bb:CC и AABB:CC совпадали.
type WiFiCheck struct{}

func NewWiFiCheck() *WiFiCheck { return &WiFiCheck{} }

func (c *WiFiCheck) Name() string { return policy.MechanismWiFi }

func (c *WiFiCheck) Check(
	_ context.Context,
	cfg policy.MechanismsConfig,
	input policy.CheckInput,
) (policy.CheckResult, error) {
	if !cfg.WiFi.Enabled {
		return skippedResult(policy.MechanismWiFi, policy.ReasonDisabled, nil), nil
	}
	if input.ClientBSSID == nil || normalizeBSSID(*input.ClientBSSID) == "" {
		return skippedResult(policy.MechanismWiFi, policy.ReasonNoClientData, nil), nil
	}

	var allowed []string
	if cfg.WiFi.RequiredBSSIDsFromClassroom {
		allowed = append(allowed, input.AllowedBSSIDs...)
	}
	allowed = append(allowed, cfg.WiFi.ExtraBSSIDs...)

	allowedNorm := make([]string, 0, len(allowed))
	for _, b := range allowed {
		n := normalizeBSSID(b)
		if n != "" {
			allowedNorm = append(allowedNorm, n)
		}
	}
	if len(allowedNorm) == 0 {
		return skippedResult(policy.MechanismWiFi, policy.ReasonNoAllowed, nil), nil
	}

	actual := normalizeBSSID(*input.ClientBSSID)
	status := policy.StatusFailed
	for _, b := range allowedNorm {
		if b == actual {
			status = policy.StatusPassed
			break
		}
	}

	return policy.CheckResult{
		Mechanism: policy.MechanismWiFi,
		Status:    status,
		Details: map[string]any{
			"expected_bssids": allowedNorm,
			"actual_bssid":    actual,
		},
	}, nil
}

func normalizeBSSID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}
