package attendance

import (
	"time"

	"github.com/google/uuid"
)

// CheckStatus — результат работы одного механизма защиты.
//
//	passed  — проверка пройдена;
//	failed  — проверка провалена (но отметка принимается, решает преподаватель);
//	skipped — проверка не была выполнена (например, клиент не прислал Wi-Fi).
type CheckStatus string

const (
	CheckPassed  CheckStatus = "passed"
	CheckFailed  CheckStatus = "failed"
	CheckSkipped CheckStatus = "skipped"
)

func (c CheckStatus) Valid() bool {
	switch c {
	case CheckPassed, CheckFailed, CheckSkipped:
		return true
	}
	return false
}

// CheckResult — персистентный результат одной проверки (security_check_results
// в БД). Привязан к одной Record через AttendanceID.
type CheckResult struct {
	ID           uuid.UUID
	AttendanceID uuid.UUID
	Mechanism    string // qr_ttl | geo | wifi | ...
	Status       CheckStatus
	Details      map[string]any // сырые данные проверки (distance_m, actual_bssid, ...)
	CheckedAt    time.Time
}
