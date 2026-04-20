package httperr

import (
	"errors"
	"log/slog"
	"net/http"
)

// Mapping — как доменную ошибку показать клиенту.
//
// Message == "" означает «использовать err.Error()» — полезно для ошибок
// с дополнительным контекстом (например, `policy.ErrInvalidConfig` обычно
// wrap'нут в fmt.Errorf с деталями вроде `ttl_seconds must be in [3, 120]`).
type Mapping struct {
	Status  int
	Code    string
	Message string
}

// matcher — одна запись реестра. Порядок регистрации важен: первый
// errors.Is-match выигрывает, это гарантирует предсказуемость, когда
// одна ошибка wrap'нет несколько sentinel'ов.
type matcher struct {
	target  error
	mapping Mapping
}

// registry — плоский slice. Заполняется ОДНОКРАТНО при инициализации
// приложения через Register (см. httperr/registry.go). Никто больше не
// должен трогать эту переменную в рантайме.
var registry []matcher

// Register добавляет маппинг в конец реестра. Вызывается из
// централизованной функции RegisterAll при сборке HTTP-слоя.
func Register(target error, m Mapping) {
	if target == nil {
		return
	}
	registry = append(registry, matcher{target: target, mapping: m})
}

// Resolve возвращает первый подходящий маппинг.
func Resolve(err error) (Mapping, bool) {
	if err == nil {
		return Mapping{}, false
	}
	for _, m := range registry {
		if errors.Is(err, m.target) {
			return m.mapping, true
		}
	}
	return Mapping{}, false
}

// RespondError — единая точка выхода для доменных ошибок в хендлерах.
//
// Поведение:
//   - err == nil → no-op;
//   - err в реестре → пишем Mapping.Status + Code + Message (или err.Error(),
//     если Message пустой);
//   - err неизвестен → 500 + логируем с request_id.
//
// 5xx-ответы всегда логируются как unexpected, 4xx — нет (это нормальный
// клиентский трафик, не хотим шумить в логи).
func RespondError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	if err == nil {
		return
	}
	m, ok := Resolve(err)
	if !ok {
		LogUnexpected(log, r, err)
		Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	msg := m.Message
	if msg == "" {
		msg = err.Error()
	}
	if m.Status >= 500 {
		LogUnexpected(log, r, err)
	}
	Write(w, m.Status, m.Code, msg)
}
