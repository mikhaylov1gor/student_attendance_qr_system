package user

import "errors"

var (
	// ErrNotFound — пользователь не найден.
	ErrNotFound = errors.New("user: not found")
	// ErrEmailTaken — email уже используется другим пользователем.
	ErrEmailTaken = errors.New("user: email already taken")
	// ErrInvalidRole — значение роли не соответствует допустимому набору.
	ErrInvalidRole = errors.New("user: invalid role")
	// ErrFullNameRequired — Last/First обязательны.
	ErrFullNameRequired = errors.New("user: last and first names are required")
	// ErrFullNameTooLong — одна из частей ФИО превышает 64 символа.
	ErrFullNameTooLong = errors.New("user: full name part too long")
	// ErrRoleGroupMismatch — нарушен инвариант «group только у студента».
	ErrRoleGroupMismatch = errors.New("user: current_group_id allowed only for students")
)
