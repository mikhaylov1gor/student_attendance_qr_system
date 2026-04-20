package policy

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CheckStatus — результат одной проверки.
// Повторяет attendance.CheckStatus на уровне значений, но домены разные:
// здесь это runtime-результат, там — персистентная сущность.
// Маппинг — задача application-слоя.
type CheckStatus string

const (
	StatusPassed  CheckStatus = "passed"
	StatusFailed  CheckStatus = "failed"
	StatusSkipped CheckStatus = "skipped"
)

// CheckInput — контекст, в котором механизм выполняет проверку.
// Содержит всё, что может пригодиться любому из существующих или будущих
// механизмов. Конкретная проверка читает только нужные поля.
type CheckInput struct {
	SessionID       uuid.UUID
	ClassroomID     *uuid.UUID
	ClassroomLat    float64
	ClassroomLng    float64
	ClassroomRadius int
	AllowedBSSIDs   []string

	// Из QR-токена
	TokenCounter   int
	CurrentCounter int
	TokenIssuedAt  time.Time

	// От клиента
	ClientGeoLat *float64
	ClientGeoLng *float64
	ClientBSSID  *string
	ClientTime   time.Time
}

// CheckResult — runtime-результат одной проверки.
// Mechanism — стабильное машинное имя (qr_ttl, geo, wifi, ...). Details — любые
// сырые данные, полезные для forensics; сериализуются в jsonb.
type CheckResult struct {
	Mechanism string
	Status    CheckStatus
	Details   map[string]any
}

// SecurityCheck — порт подключаемого механизма защиты (Strategy).
// Регистрация конкретных реализаций — явный слайс в composition root.
type SecurityCheck interface {
	Name() string
	Check(ctx context.Context, cfg MechanismsConfig, input CheckInput) (CheckResult, error)
}
