// Package audit содержит доменные типы для tamper-evident журнала действий.
// Реализация hash-chain и canonical JSON — в application-слое (этап 7).
package audit

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// Action — стабильный машинный идентификатор действия в журнале.
// Тип string, чтобы расширение не требовало миграции БД.
type Action string

// Набор кодов действий. Константы растут вместе с функциональностью.
// Добавлять новое действие = добавить константу и использовать её в сервисе.
const (
	ActionLoginSuccess   Action = "login_success"
	ActionLoginFailed    Action = "login_failed"
	ActionLogout         Action = "logout"
	ActionTokenRefreshed Action = "token_refreshed"

	ActionUserCreated       Action = "user_created"
	ActionUserUpdated       Action = "user_updated"
	ActionUserDeleted       Action = "user_deleted"
	ActionUserPasswordReset Action = "user_password_reset"

	ActionPolicyCreated    Action = "policy_created"
	ActionPolicyUpdated    Action = "policy_updated"
	ActionPolicyDeleted    Action = "policy_deleted"
	ActionPolicyDefaultSet Action = "policy_default_set"

	ActionSessionCreated Action = "session_created"
	ActionSessionUpdated Action = "session_updated"
	ActionSessionStarted Action = "session_started"
	ActionSessionClosed  Action = "session_closed"
	ActionSessionDeleted Action = "session_deleted"

	ActionAttendanceSubmitted Action = "attendance_submitted"
	ActionAttendanceResolved  Action = "attendance_resolved"

	ActionCatalogCourseCreated    Action = "course_created"
	ActionCatalogCourseUpdated    Action = "course_updated"
	ActionCatalogCourseDeleted    Action = "course_deleted"
	ActionCatalogGroupCreated     Action = "group_created"
	ActionCatalogGroupUpdated     Action = "group_updated"
	ActionCatalogGroupDeleted     Action = "group_deleted"
	ActionCatalogStreamCreated    Action = "stream_created"
	ActionCatalogStreamUpdated    Action = "stream_updated"
	ActionCatalogStreamDeleted    Action = "stream_deleted"
	ActionCatalogClassroomCreated Action = "classroom_created"
	ActionCatalogClassroomUpdated Action = "classroom_updated"
	ActionCatalogClassroomDeleted Action = "classroom_deleted"
)

// HashLen — длина SHA-256 в байтах (и для prev_hash, и для record_hash).
const HashLen = 32

// Entry — запись в журнале аудита.
//
// PrevHash и RecordHash — бинарные, длиной HashLen. Genesis-запись имеет
// PrevHash = bytes32(0). RecordHash = SHA-256(PrevHash ‖ canonical_payload),
// где canonical_payload вычисляется детерминированно (см. этап 7).
//
// ID присваивается базой (bigserial). Используется как источник порядка
// записей в цепочке — надёжнее uuid.
type Entry struct {
	ID         int64
	PrevHash   []byte
	RecordHash []byte
	OccurredAt time.Time
	ActorID    *uuid.UUID
	ActorRole  string
	Action     Action
	EntityType string
	EntityID   string
	Payload    map[string]any
	IPAddress  net.IP
	UserAgent  string
}
