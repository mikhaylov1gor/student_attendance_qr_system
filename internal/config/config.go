package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTPPort   int    `env:"HTTP_PORT" envDefault:"8080"`
	AppVersion string `env:"APP_VERSION" envDefault:"dev"`

	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"text"`

	DatabaseDSN string `env:"DATABASE_DSN"`

	// PIIEncryptionKeyB64 — 32 байта в base64 (openssl rand -base64 32).
	// Используется для шифрования ФИО в колонках users.full_name_ciphertext.
	PIIEncryptionKeyB64 string `env:"PII_ENCRYPTION_KEY"`

	// Argon2id — OWASP 2024 baseline.
	Argon2MemoryKiB   uint32 `env:"ARGON2_MEMORY_KIB"  envDefault:"65536"`
	Argon2Iterations  uint32 `env:"ARGON2_ITERATIONS"  envDefault:"3"`
	Argon2Parallelism uint8  `env:"ARGON2_PARALLELISM" envDefault:"2"`

	// JWT — HS256 access-токен. Refresh-токен случайный, подписи не требует.
	JWTAccessSecretB64 string        `env:"JWT_ACCESS_SECRET"`
	JWTIssuer          string        `env:"JWT_ISSUER"        envDefault:"attendance-api"`
	JWTAccessTTL       time.Duration `env:"JWT_ACCESS_TTL"    envDefault:"15m"`
	JWTRefreshTTL      time.Duration `env:"JWT_REFRESH_TTL"   envDefault:"168h"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	return cfg, nil
}
