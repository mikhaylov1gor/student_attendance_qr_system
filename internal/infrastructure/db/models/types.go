// Package models содержит Gorm-модели, изолированные от доменных типов.
// Преобразование domain ↔ model выполняется явными mapper-функциями в этом
// же пакете (см. mapper.go).
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONB — общий контейнер для JSONB-колонок.
// Хранит сырые bytes; парсинг в типизированную структуру — задача mapper'а.
type JSONB []byte

// Value пишется в Postgres как string (Postgres jsonb принимает string).
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan читает []byte или string.
func (j *JSONB) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		cp := make([]byte, len(v))
		copy(cp, v)
		*j = cp
	case string:
		*j = []byte(v)
	default:
		return fmt.Errorf("JSONB: unsupported scan type %T", src)
	}
	return nil
}

// MarshalJSON / UnmarshalJSON делают тип прозрачным в json.Encoder.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}

// MarshalToJSONB сериализует любое значение в JSONB.
func MarshalToJSONB(v any) (JSONB, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal jsonb: %w", err)
	}
	return JSONB(b), nil
}

// UnmarshalFromJSONB десериализует JSONB в указатель на destination.
func UnmarshalFromJSONB(j JSONB, dst any) error {
	if len(j) == 0 {
		return nil
	}
	if err := json.Unmarshal(j, dst); err != nil {
		return fmt.Errorf("unmarshal jsonb: %w", err)
	}
	return nil
}
