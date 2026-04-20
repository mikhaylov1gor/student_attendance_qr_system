package checks_test

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"

	"attendance/internal/domain/policy"
	"attendance/internal/domain/policy/checks"
)

// Известные пары координат для валидации haversine.
// (Москва — Санкт-Петербург: ~633 км; ТГУ главный корпус → ул. Ленина 34 Томск: ~110 м).
func TestHaversineMeters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		wantMeters             float64
		tolerancePercent       float64
	}{
		{"one point with itself", 56.469849, 84.948042, 56.469849, 84.948042, 0, 0.01},
		// ТГУ → точка в ~500 м севернее
		{"ТГУ vs point ~500m north", 56.469849, 84.948042, 56.474349, 84.948042, 500, 5},
		// Москва → СПб ≈ 633 км
		{"Москва — Санкт-Петербург", 55.7558, 37.6173, 59.9343, 30.3351, 633200, 5},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checks.HaversineMeters(tc.lat1, tc.lon1, tc.lat2, tc.lon2)
			if tc.wantMeters == 0 {
				if got > 0.01 {
					t.Fatalf("same point: got %.4f m, expected 0", got)
				}
				return
			}
			diff := math.Abs(got - tc.wantMeters)
			allowed := tc.wantMeters * tc.tolerancePercent / 100
			if diff > allowed {
				t.Fatalf("distance = %.2f m, want %.2f ± %.2f m", got, tc.wantMeters, allowed)
			}
		})
	}
}

func TestGeoCheck(t *testing.T) {
	t.Parallel()

	c := checks.NewGeoCheck()
	classroomID := uuid.New()

	lat := 56.469849
	lng := 84.948042

	base := policy.CheckInput{
		ClassroomID:     &classroomID,
		ClassroomLat:    lat,
		ClassroomLng:    lng,
		ClassroomRadius: 25,
	}

	cfgEnabled := policy.MechanismsConfig{Geo: policy.GeoConfig{Enabled: true}}
	cfgDisabled := policy.MechanismsConfig{Geo: policy.GeoConfig{Enabled: false}}

	withClient := func(in policy.CheckInput, lat, lng float64) policy.CheckInput {
		in.ClientGeoLat = &lat
		in.ClientGeoLng = &lng
		return in
	}

	t.Run("disabled → skipped", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfgDisabled, withClient(base, lat, lng))
		if res.Status != policy.StatusSkipped {
			t.Fatalf("status = %s, want skipped", res.Status)
		}
		if res.Details["reason"] != policy.ReasonDisabled {
			t.Errorf("reason = %v, want disabled", res.Details["reason"])
		}
	})

	t.Run("no classroom → skipped", func(t *testing.T) {
		t.Parallel()
		in := withClient(base, lat, lng)
		in.ClassroomID = nil
		res, _ := c.Check(context.Background(), cfgEnabled, in)
		if res.Status != policy.StatusSkipped {
			t.Fatalf("status = %s, want skipped", res.Status)
		}
		if res.Details["reason"] != policy.ReasonNoClassroom {
			t.Errorf("reason = %v, want no_classroom", res.Details["reason"])
		}
	})

	t.Run("no client data → skipped", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfgEnabled, base)
		if res.Status != policy.StatusSkipped {
			t.Fatalf("status = %s, want skipped", res.Status)
		}
		if res.Details["reason"] != policy.ReasonNoClientData {
			t.Errorf("reason = %v, want no_client_data", res.Details["reason"])
		}
	})

	t.Run("внутри радиуса → passed", func(t *testing.T) {
		t.Parallel()
		res, _ := c.Check(context.Background(), cfgEnabled, withClient(base, lat, lng))
		if res.Status != policy.StatusPassed {
			t.Fatalf("status = %s, want passed", res.Status)
		}
	})

	t.Run("вне радиуса → failed", func(t *testing.T) {
		t.Parallel()
		// ~500 м севернее — явно за 25-метровым радиусом.
		res, _ := c.Check(context.Background(), cfgEnabled, withClient(base, lat+0.0045, lng))
		if res.Status != policy.StatusFailed {
			t.Fatalf("status = %s, want failed", res.Status)
		}
		if res.Details["distance_m"] == nil {
			t.Errorf("details.distance_m not set: %+v", res.Details)
		}
	})

	t.Run("radius override из политики", func(t *testing.T) {
		t.Parallel()
		big := 600 // override
		cfg := policy.MechanismsConfig{Geo: policy.GeoConfig{Enabled: true, RadiusOverrideM: &big}}
		res, _ := c.Check(context.Background(), cfg, withClient(base, lat+0.0045, lng))
		if res.Status != policy.StatusPassed {
			t.Fatalf("status = %s, want passed (override should expand radius)", res.Status)
		}
	})
}
