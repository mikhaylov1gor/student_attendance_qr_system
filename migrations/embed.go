package migrations

import "embed"

// FS содержит все SQL-миграции, встроенные в бинарь.
//
//go:embed *.sql
var FS embed.FS
