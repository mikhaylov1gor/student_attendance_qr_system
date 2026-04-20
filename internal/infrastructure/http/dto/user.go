package dto

import (
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain/user"
)

// UserResponse — открытое представление пользователя для admin-эндпоинтов.
// Password-хэш, естественно, не отдаём.
type UserResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	FullName       string    `json:"full_name"`
	Role           string    `json:"role"`
	CurrentGroupID *string   `json:"current_group_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func UserFromDomain(u user.User) UserResponse {
	var gid *string
	if u.CurrentGroupID != nil {
		v := u.CurrentGroupID.String()
		gid = &v
	}
	return UserResponse{
		ID:             u.ID.String(),
		Email:          u.Email,
		FullName:       u.FullName.String(),
		Role:           string(u.Role),
		CurrentGroupID: gid,
		CreatedAt:      u.CreatedAt,
	}
}

// UserListResponse — GET /users.
type UserListResponse struct {
	Items []UserResponse `json:"items"`
	Total int            `json:"total"`
}

// CreateUserRequest — POST /users.
// Password опциональный: если пуст, сервис генерирует temp-пароль и возвращает
// в поле TempPassword ответа.
type CreateUserRequest struct {
	Email    string     `json:"email"           validate:"required,email,max=254"`
	Password string     `json:"password,omitempty" validate:"omitempty,min=8,max=512"`
	Role     string     `json:"role"            validate:"required,oneof=student teacher admin"`
	Last     string     `json:"last"            validate:"required,min=1,max=64"`
	First    string     `json:"first"           validate:"required,min=1,max=64"`
	Middle   string     `json:"middle,omitempty" validate:"omitempty,max=64"`
	GroupID  *uuid.UUID `json:"current_group_id,omitempty"`
}

// CreateUserResponse — тело ответа POST /users.
// TempPassword заполняется только если сервис сам сгенерировал пароль.
type CreateUserResponse struct {
	User         UserResponse `json:"user"`
	TempPassword string       `json:"temp_password,omitempty"`
}

// UpdateUserRequest — PATCH /users/:id.
// Разделение group_id/clear_group — как в session.PATCH (см. объяснение там):
// чтобы отличить «поле не прислано» от «снять привязку».
type UpdateUserRequest struct {
	Email      *string    `json:"email,omitempty"   validate:"omitempty,email,max=254"`
	Role       *string    `json:"role,omitempty"    validate:"omitempty,oneof=student teacher admin"`
	Last       *string    `json:"last,omitempty"    validate:"omitempty,min=1,max=64"`
	First      *string    `json:"first,omitempty"   validate:"omitempty,min=1,max=64"`
	Middle     *string    `json:"middle,omitempty"  validate:"omitempty,max=64"`
	GroupID    *uuid.UUID `json:"current_group_id,omitempty"`
	ClearGroup bool       `json:"clear_group,omitempty"`
}

// ResetPasswordResponse — тело ответа POST /users/:id/reset-password.
type ResetPasswordResponse struct {
	TempPassword string `json:"temp_password"`
}
