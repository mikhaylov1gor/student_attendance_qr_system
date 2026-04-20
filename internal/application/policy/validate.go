// Package policy содержит use case'ы управления политиками безопасности.
package policy

import (
	"errors"
	"fmt"

	domainpolicy "attendance/internal/domain/policy"
	"attendance/internal/domain/session"
)

// ValidateMechanisms выполняет sanity-check конфига механизмов защиты.
// Вызывается в Create и Update. Runtime-поведение (skipped при пустом
// allowlist и т.п.) валидатор не отключает — он ловит только очевидный brak.
func ValidateMechanisms(m domainpolicy.MechanismsConfig) error {
	var errs []error

	if m.QRTTL.Enabled {
		ttl := m.QRTTL.TTLSeconds
		if ttl < session.MinQRTTLSeconds || ttl > session.MaxQRTTLSeconds {
			errs = append(errs, fmt.Errorf(
				"qr_ttl.ttl_seconds must be in [%d, %d], got %d",
				session.MinQRTTLSeconds, session.MaxQRTTLSeconds, ttl,
			))
		}
	}

	if m.Geo.Enabled && m.Geo.RadiusOverrideM != nil {
		if *m.Geo.RadiusOverrideM <= 0 {
			errs = append(errs, fmt.Errorf("geo.radius_override_m must be > 0"))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", domainpolicy.ErrInvalidConfig, errors.Join(errs...))
	}
	return nil
}
