// Package repo — реализации доменных Repository-портов поверх Gorm.
// Тут же — хелперы распознавания частых Postgres-ошибок и проброс их в
// доменные sentinel'ы.
package repo

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgCodeUnique — SQLSTATE 23505 (unique_violation).
const pgCodeUnique = "23505"

// isUniqueViolation возвращает true, если err — нарушение unique.
// Если constraint непустая строка, дополнительно сверяем имя констрейнта.
func isUniqueViolation(err error, constraint string) bool {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return false
	}
	if pg.Code != pgCodeUnique {
		return false
	}
	if constraint == "" {
		return true
	}
	return pg.ConstraintName == constraint
}
