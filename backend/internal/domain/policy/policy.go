// Package policy содержит конфигурацию механизмов защиты и порты для
// подключаемых проверок (Strategy). Сами реализации QRTTLCheck/GeoCheck/WiFiCheck
// живут в подпакете checks (этап 6).
package policy

import (
	"time"

	"github.com/google/uuid"
)

// MechanismsConfig — типизированный аналог JSONB в security_policies.mechanisms.
// Добавление нового механизма = расширение структуры + новая проверка + апдейт
// JSON-сериализации на инфра-уровне. Миграция БД не требуется.
type MechanismsConfig struct {
	QRTTL           QRTTLConfig  `json:"qr_ttl"`
	Geo             GeoConfig    `json:"geo"`
	WiFi            WiFiConfig   `json:"wifi"`
	BluetoothBeacon BeaconConfig `json:"bluetooth_beacon"`
}

// QRTTLConfig — параметры проверки свежести QR по счётчику ротации.
type QRTTLConfig struct {
	Enabled    bool `json:"enabled"`
	TTLSeconds int  `json:"ttl_seconds"`
}

// GeoConfig — параметры геопроверки. Координаты берутся из classroom;
// RadiusOverrideM позволяет переопределить radius_m на уровне политики.
type GeoConfig struct {
	Enabled         bool `json:"enabled"`
	RadiusOverrideM *int `json:"radius_override_m,omitempty"`
}

// WiFiConfig — параметры Wi-Fi проверки. Если RequiredBSSIDsFromClassroom=true,
// сервис подставляет allowed_bssids из classroom.
type WiFiConfig struct {
	Enabled                     bool     `json:"enabled"`
	RequiredBSSIDsFromClassroom bool     `json:"required_bssids_from_classroom,omitempty"`
	ExtraBSSIDs                 []string `json:"extra_bssids,omitempty"`
}

// BeaconConfig — заглушка под будущую проверку Bluetooth-маячка.
type BeaconConfig struct {
	Enabled bool `json:"enabled"`
}

// SecurityPolicy — именованный набор параметров MechanismsConfig.
// IsDefault: ровно одна запись в БД имеет true (partial unique index).
type SecurityPolicy struct {
	ID         uuid.UUID
	Name       string
	Mechanisms MechanismsConfig
	IsDefault  bool
	CreatedBy  *uuid.UUID
	CreatedAt  time.Time
	DeletedAt  *time.Time
}
