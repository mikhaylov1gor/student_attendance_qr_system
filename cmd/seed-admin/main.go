// Команда seed-admin создаёт пользователя любой роли в БД.
//
// Несмотря на исторически закрепившееся имя, инструмент создаёт не только
// администраторов: --role позволяет выбрать admin (по умолчанию), teacher или
// student. До появления полноценного REST для admin'ского создания пользователей
// (этап 10) — это единственный способ получить teacher/student аккаунт для
// тестирования attendance-flow.
//
// Примеры:
//
//	go run ./cmd/seed-admin --email admin@tsu.ru --password '...' --last A --first B
//	go run ./cmd/seed-admin --role teacher --email t@tsu.ru --password '...' --last C --first D
//	go run ./cmd/seed-admin --role student --group-id <uuid> --email s@tsu.ru --password '...' --last E --first F
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"attendance/internal/config"
	"attendance/internal/domain/user"
	"attendance/internal/infrastructure/crypto"
	"attendance/internal/infrastructure/db"
	"attendance/internal/infrastructure/db/repo"
	"attendance/internal/platform/logging"
)

func main() {
	log.SetFlags(0)
	_ = godotenv.Load()

	email := flag.String("email", "", "email пользователя (обязательно)")
	password := flag.String("password", "", "пароль (обязательно)")
	last := flag.String("last", "", "фамилия (обязательно)")
	first := flag.String("first", "", "имя (обязательно)")
	middle := flag.String("middle", "", "отчество (опционально)")
	roleStr := flag.String("role", "admin", "роль: admin | teacher | student")
	groupIDStr := flag.String("group-id", "", "uuid группы (только для role=student)")
	flag.Parse()

	if *email == "" || *password == "" || *last == "" || *first == "" {
		log.Println("требуются флаги --email, --password, --last, --first")
		flag.Usage()
		os.Exit(2)
	}

	role := user.Role(*roleStr)
	if !role.Valid() {
		log.Fatalf("неизвестная роль %q (admin|teacher|student)", *roleStr)
	}
	var groupID *uuid.UUID
	if *groupIDStr != "" {
		gid, err := uuid.Parse(*groupIDStr)
		if err != nil {
			log.Fatalf("--group-id: %v", err)
		}
		groupID = &gid
	}
	if role == user.RoleStudent && groupID == nil {
		log.Fatal("для role=student обязателен --group-id")
	}
	if role != user.RoleStudent && groupID != nil {
		log.Fatal("--group-id допустим только для role=student")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.DatabaseDSN == "" {
		log.Fatal("DATABASE_DSN пуст; проверь .env")
	}
	if cfg.PIIEncryptionKeyB64 == "" {
		log.Fatal("PII_ENCRYPTION_KEY пуст; сгенерируй: openssl rand -base64 32")
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat, os.Stderr)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	gormDB, err := db.Open(ctx, cfg.DatabaseDSN, logger, db.DefaultOptions())
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close(gormDB)

	enc, err := crypto.NewAESGCMEncryptorFromBase64(cfg.PIIEncryptionKeyB64)
	if err != nil {
		log.Fatalf("init encryptor (ожидается 32 байта после base64): %v", err)
	}

	hasher := crypto.NewArgon2idHasher(crypto.Argon2idParams{
		Memory:      cfg.Argon2MemoryKiB,
		Iterations:  cfg.Argon2Iterations,
		Parallelism: cfg.Argon2Parallelism,
	})

	passwordHash, err := hasher.Hash(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	fullName, err := user.NewFullName(*last, *first, *middle)
	if err != nil {
		log.Fatalf("full name: %v", err)
	}

	now := time.Now().UTC()
	u := user.User{
		ID:             uuid.New(),
		Email:          *email,
		PasswordHash:   passwordHash,
		FullName:       fullName,
		Role:           role,
		CurrentGroupID: groupID,
		CreatedAt:      now,
	}

	userRepo := repo.NewUserRepo(gormDB, enc)

	if err := userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, user.ErrEmailTaken) {
			log.Fatalf("пользователь с email %q уже существует", u.Email)
		}
		log.Fatalf("create user: %v", err)
	}

	// Читаем обратно — проверяем, что шифрование обратимо и ФИО корректно
	// восстанавливается.
	back, err := userRepo.GetByEmail(ctx, u.Email)
	if err != nil {
		log.Fatalf("read back: %v", err)
	}

	ok, err := hasher.Verify(*password, back.PasswordHash)
	if err != nil || !ok {
		log.Fatalf("verify password: ok=%v err=%v", ok, err)
	}

	fmt.Printf("✓ user создан (role=%s)\n", back.Role)
	fmt.Printf("  id:     %s\n", back.ID)
	fmt.Printf("  email:  %s\n", back.Email)
	fmt.Printf("  ФИО:    %s (расшифровано из bytea)\n", back.FullName)
	if back.CurrentGroupID != nil {
		fmt.Printf("  group:  %s\n", *back.CurrentGroupID)
	}
	fmt.Printf("  hash:   %s\n", truncate(back.PasswordHash, 64))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
