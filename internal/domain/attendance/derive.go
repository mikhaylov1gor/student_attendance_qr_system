package attendance

import "attendance/internal/domain/policy"

// DerivePreliminaryStatus вычисляет preliminary_status по сводке результатов
// механизмов защиты.
//
// Политика non-blocking (см. memory project_security_mechanisms_policy):
//   - любой failed → needs_review (преподаватель решает вручную);
//   - все passed/skipped → accepted.
//
// StatusRejected здесь никогда не возвращается — rejected появляется только
// как final_status при ручном решении преподавателя. Инвариант закрепляет
// CHECK attendance_final_status_check в миграции 0001.
func DerivePreliminaryStatus(results []policy.CheckResult) Status {
	for _, r := range results {
		if r.Status == policy.StatusFailed {
			return StatusNeedsReview
		}
	}
	return StatusAccepted
}
