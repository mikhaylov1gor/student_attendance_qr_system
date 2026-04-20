// Package user описывает доменные типы и порт репозитория для пользователей
// системы (студент, преподаватель, администратор).
package user

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Role — роль пользователя. Соответствует ENUM user_role в БД.
type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

// Valid возвращает true, если значение является одним из разрешённых.
func (r Role) Valid() bool {
	switch r {
	case RoleStudent, RoleTeacher, RoleAdmin:
		return true
	}
	return false
}

func (r Role) String() string { return string(r) }

// FullName — value object «ФИО». В БД хранится в зашифрованном виде (AES-GCM).
// В домене оперируем plaintext-значением, шифрование/дешифрование — задача
// крипто-порта (FieldEncryptor) и mapper'а инфраструктуры.
type FullName struct {
	Last   string
	First  string
	Middle string // отчество, опционально
}

// NewFullName конструирует FullName с валидацией.
func NewFullName(last, first, middle string) (FullName, error) {
	last = strings.TrimSpace(last)
	first = strings.TrimSpace(first)
	middle = strings.TrimSpace(middle)

	if last == "" || first == "" {
		return FullName{}, ErrFullNameRequired
	}
	for _, s := range []string{last, first, middle} {
		if utf8.RuneCountInString(s) > 64 {
			return FullName{}, ErrFullNameTooLong
		}
	}
	return FullName{Last: last, First: first, Middle: middle}, nil
}

// String возвращает «Фамилия Имя Отчество» (отчество опускается, если пусто).
func (n FullName) String() string {
	parts := []string{n.Last, n.First}
	if n.Middle != "" {
		parts = append(parts, n.Middle)
	}
	return strings.Join(parts, " ")
}

// User — доменная сущность пользователя.
//
// PasswordHash — self-describing argon2id строка (см. crypto-порт PasswordHasher).
// FullName — plaintext; mapper инфраструктуры шифрует/расшифровывает в пару
// (ciphertext, nonce).
// CurrentGroupID заполнен только для студентов (см. invariant users_role_group_check
// в миграции 0001).
type User struct {
	ID             uuid.UUID
	Email          string
	PasswordHash   string
	FullName       FullName
	Role           Role
	CurrentGroupID *uuid.UUID
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

// IsDeleted — true, если запись помечена soft-delete.
func (u User) IsDeleted() bool { return u.DeletedAt != nil }
