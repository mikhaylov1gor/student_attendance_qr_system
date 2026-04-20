package checks

import (
	"context"
	"math"

	"attendance/internal/domain/policy"
)

// GeoCheck — проверка геолокации студента.
//
// Всегда skipped, если:
//   - отключено в политике (reason=disabled);
//   - у сессии нет classroom — онлайн-формат (reason=no_classroom);
//   - клиент не прислал координаты (reason=no_client_data).
//
// В противном случае — haversine-расстояние между classroom и client
// сравнивается с radius_m (или override из политики).
type GeoCheck struct{}

func NewGeoCheck() *GeoCheck { return &GeoCheck{} }

func (c *GeoCheck) Name() string { return policy.MechanismGeo }

func (c *GeoCheck) Check(
	_ context.Context,
	cfg policy.MechanismsConfig,
	input policy.CheckInput,
) (policy.CheckResult, error) {
	if !cfg.Geo.Enabled {
		return skippedResult(policy.MechanismGeo, policy.ReasonDisabled, nil), nil
	}
	if input.ClassroomID == nil {
		return skippedResult(policy.MechanismGeo, policy.ReasonNoClassroom, nil), nil
	}
	if input.ClientGeoLat == nil || input.ClientGeoLng == nil {
		return skippedResult(policy.MechanismGeo, policy.ReasonNoClientData, nil), nil
	}

	radius := input.ClassroomRadius
	if cfg.Geo.RadiusOverrideM != nil {
		radius = *cfg.Geo.RadiusOverrideM
	}

	distance := HaversineMeters(
		input.ClassroomLat, input.ClassroomLng,
		*input.ClientGeoLat, *input.ClientGeoLng,
	)

	status := policy.StatusFailed
	if distance <= float64(radius) {
		status = policy.StatusPassed
	}

	return policy.CheckResult{
		Mechanism: policy.MechanismGeo,
		Status:    status,
		Details: map[string]any{
			"expected_lat": input.ClassroomLat,
			"expected_lng": input.ClassroomLng,
			"actual_lat":   *input.ClientGeoLat,
			"actual_lng":   *input.ClientGeoLng,
			"distance_m":   roundFloat(distance, 2),
			"radius_m":     radius,
		},
	}, nil
}

// HaversineMeters возвращает расстояние между двумя геокоординатами в метрах
// по формуле haversine. Радиус Земли — 6371000 м.
func HaversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }

	phi1 := toRad(lat1)
	phi2 := toRad(lat2)
	dphi := toRad(lat2 - lat1)
	dlam := toRad(lon2 - lon1)

	a := math.Sin(dphi/2)*math.Sin(dphi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(dlam/2)*math.Sin(dlam/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

func roundFloat(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
