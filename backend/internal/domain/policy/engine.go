package policy

import (
	"context"
	"fmt"
)

// Engine — оркестратор Security Checks. Stateless, thread-safe.
//
// Регистрация чеков происходит при сборке в composition root:
//
//	engine := policy.NewEngine(
//	    checks.NewQRTTLCheck(),
//	    checks.NewGeoCheck(),
//	    checks.NewWiFiCheck(),
//	)
//
// Добавление нового механизма = одна регистрация + одна секция
// в MechanismsConfig. Evaluate менять не нужно.
type Engine struct {
	checks []SecurityCheck
}

// NewEngine сохраняет порядок переданных чеков — он же будет порядком
// результатов в Evaluate (важно для стабильности тестов и логов).
func NewEngine(checks ...SecurityCheck) *Engine {
	cp := make([]SecurityCheck, len(checks))
	copy(cp, checks)
	return &Engine{checks: cp}
}

// Checks возвращает копию списка зарегистрированных чеков (для диагностики,
// health-эндпоинтов, тестов).
func (e *Engine) Checks() []SecurityCheck {
	cp := make([]SecurityCheck, len(e.checks))
	copy(cp, e.checks)
	return cp
}

// Evaluate прогоняет все чеки в порядке регистрации.
//
// Семантика:
//   - каждый чек сам решает, что вернуть при disabled / отсутствии данных
//     (обычно StatusSkipped с заполненным reason в Details);
//   - инфраструктурная ошибка (err != nil) ОСТАНАВЛИВАЕТ прогон. Это
//     отличается от provider-логики «механизм не смог проверить» — такую
//     ситуацию чек маппит в StatusSkipped/StatusFailed сам;
//   - если ctx отменён — возвращаем ctx.Err() без завершения оставшихся чеков.
func (e *Engine) Evaluate(
	ctx context.Context,
	cfg MechanismsConfig,
	input CheckInput,
) ([]CheckResult, error) {
	results := make([]CheckResult, 0, len(e.checks))
	for _, c := range e.checks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		r, err := c.Check(ctx, cfg, input)
		if err != nil {
			return nil, fmt.Errorf("policy: check %q: %w", c.Name(), err)
		}
		results = append(results, r)
	}
	return results, nil
}
