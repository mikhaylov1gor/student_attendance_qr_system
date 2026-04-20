package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"

	"attendance/internal/config"
	"attendance/migrations"
)

const usage = `goose-обёртка для миграций БД.

Использование:
  migrate <command> [args]

Команды:
  up                 применить все новые миграции
  up-by-one          применить одну следующую миграцию
  down               откатить последнюю миграцию
  reset              откатить все миграции
  status             показать статус миграций
  version            показать текущую версию
  create <name>      создать новый SQL-файл миграции в ./migrations
`

func main() {
	log.SetFlags(0)

	_ = godotenv.Load()

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	command := args[0]

	if command == "create" {
		if len(args) < 2 {
			log.Fatal("migrate create: требуется имя миграции, например: migrate create add_refresh_tokens")
		}
		if err := createMigration(args[1]); err != nil {
			log.Fatalf("migrate create: %v", err)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if strings.TrimSpace(cfg.DatabaseDSN) == "" {
		log.Fatal("DATABASE_DSN пуст; проверь .env")
	}

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set dialect: %v", err)
	}

	extraArgs := args[1:]
	if err := goose.Run(command, db, ".", extraArgs...); err != nil {
		log.Fatalf("migrate %s: %v", command, err)
	}
}

// createMigration создаёт новый пустой SQL-файл с префиксом `NNNN` в каталоге ./migrations.
// `NNNN` вычисляется как max(existing)+1. Если миграций ещё нет — `0001`.
func createMigration(name string) error {
	if name == "" {
		return errors.New("имя миграции не может быть пустым")
	}

	dir := "migrations"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("создать каталог миграций: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("читать каталог миграций: %w", err)
	}

	next := 1
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "%04d_", &n); err == nil && n >= next {
			next = n + 1
		}
	}

	safeName := strings.ReplaceAll(strings.ToLower(name), "-", "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")
	path := filepath.Join(dir, fmt.Sprintf("%04d_%s.sql", next, safeName))

	body := fmt.Sprintf(`-- Migration %04d: %s
-- created at %s

-- +goose Up
-- TODO: SQL для up-миграции

-- +goose Down
-- TODO: SQL для down-миграции
`, next, name, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("создать файл: %w", err)
	}

	fmt.Printf("создана миграция: %s\n", path)
	return nil
}
