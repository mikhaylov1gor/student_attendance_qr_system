package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// CanonicalEvent — фиксированный набор полей, которые хэшируются в hash-chain.
// Именно этот формат — точка истины для верификации: любое отличие между
// Append-написанием и Verify-пересчётом приведёт к false negative.
//
// Из цепочки сознательно исключены:
//   - id — присваивается базой (bigserial); на момент хэширования его нет;
//   - prev_hash — складывается с payload наружу (см. ComputeRecordHash);
//   - record_hash — это и есть результат.
type CanonicalEvent struct {
	OccurredAt time.Time
	ActorID    string // uuid или "" для системных событий
	ActorRole  string
	Action     string
	EntityType string
	EntityID   string
	Payload    map[string]any
	IPAddress  string // строковое представление inet или ""
	UserAgent  string
}

// Canonicalize возвращает детерминированный JSON-байт-буфер для CanonicalEvent.
//
// Инварианты:
//   - ключи отсортированы алфавитно (на всех уровнях вложенности);
//   - time → RFC3339Nano UTC (time.Time преобразуется в строку);
//   - nil/empty: "" для строк, null для отсутствующего payload;
//   - без пробелов между токенами (json.Marshal по дефолту).
//
// Реализация: roundtrip через map[string]any — encoding/json сортирует
// ключи map'ов с Go 1.12 автоматически.
func Canonicalize(e CanonicalEvent) ([]byte, error) {
	payload, err := canonicalizeValue(e.Payload)
	if err != nil {
		return nil, fmt.Errorf("canonicalize payload: %w", err)
	}
	root := map[string]any{
		"action":      e.Action,
		"actor_id":    e.ActorID,
		"actor_role":  e.ActorRole,
		"entity_id":   e.EntityID,
		"entity_type": e.EntityType,
		"ip_address":  e.IPAddress,
		"occurred_at": e.OccurredAt.UTC().Format(time.RFC3339Nano),
		"payload":     payload,
		"user_agent":  e.UserAgent,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("canonicalize encode: %w", err)
	}
	// Encoder добавляет \n в конце — обрезаем.
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

// canonicalizeValue рекурсивно нормализует любое JSON-значение.
//
// Примитивы пропускаем как есть — они уже имеют каноническое JSON-представление.
// time.Time явно форматируется в RFC3339Nano UTC.
// map'ы разворачиваем рекурсивно (encoding/json сам сортирует ключи, но мы
// дополнительно нормализуем вложенные time/struct).
// struct'ы сводим к примитивам через roundtrip json (однократный, не рекурсивный).
func canonicalizeValue(v any) (any, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	// Примитивы JSON — возвращаем как есть.
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		json.Number:
		return val, nil

	case time.Time:
		return val.UTC().Format(time.RFC3339Nano), nil

	case map[string]any:
		if val == nil { // nil-map → null, чтобы отличать от пустого {}
			return nil, nil
		}
		out := make(map[string]any, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			nv, err := canonicalizeValue(val[k])
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil

	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			nv, err := canonicalizeValue(item)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil

	default:
		// Для struct'ов, указателей, массивов кастомных типов и т.п. —
		// roundtrip через json, чтобы свести к примитиву|map|slice.
		b, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("marshal %T: %w", val, err)
		}
		var round any
		if err := json.Unmarshal(b, &round); err != nil {
			return nil, fmt.Errorf("unmarshal roundtrip %T: %w", val, err)
		}
		// Если roundtrip вернул тот же тип, что пришёл — защита от бесконечной
		// рекурсии (для кастомных строковых типов и т.п.).
		switch round.(type) {
		case map[string]any, []any:
			return canonicalizeValue(round)
		default:
			return round, nil
		}
	}
}
