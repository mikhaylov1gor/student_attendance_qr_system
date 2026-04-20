package httperr

import (
	"net/http"

	"attendance/internal/domain/attendance"
	"attendance/internal/domain/auth"
	"attendance/internal/domain/catalog"
	"attendance/internal/domain/policy"
	"attendance/internal/domain/session"
	"attendance/internal/domain/user"
)

// RegisterAll регистрирует все известные доменные sentinel-ошибки в мапперe.
// Вызывается один раз при сборке HTTP-слоя (см. cmd/api/main.go).
//
// Порядок внутри групп — от частного к общему: если где-то несколько sentinel'ов
// окажутся в цепочке `errors.Is`, победит первый зарегистрированный.
//
// Группы разнесены комментариями, чтобы при добавлении нового домена было
// очевидно, куда дописать его ошибки.
func RegisterAll() {
	// ------------------------------------------------------------------
	// auth
	// ------------------------------------------------------------------
	Register(auth.ErrInvalidCredentials, Mapping{
		Status: http.StatusUnauthorized, Code: "invalid_credentials",
		Message: "wrong email or password",
	})
	Register(auth.ErrTokenExpired, Mapping{
		Status: http.StatusUnauthorized, Code: "token_expired",
	})
	Register(auth.ErrTokenRevoked, Mapping{
		Status: http.StatusUnauthorized, Code: "token_revoked",
	})
	Register(auth.ErrInvalidToken, Mapping{
		Status: http.StatusUnauthorized, Code: "invalid_token",
	})
	Register(auth.ErrUnauthorized, Mapping{
		Status: http.StatusUnauthorized, Code: "unauthorized",
	})
	Register(auth.ErrForbidden, Mapping{
		Status: http.StatusForbidden, Code: "forbidden",
	})

	// ------------------------------------------------------------------
	// user
	// ------------------------------------------------------------------
	Register(user.ErrEmailTaken, Mapping{
		Status: http.StatusConflict, Code: "email_taken",
		Message: "email already in use",
	})
	Register(user.ErrNotFound, Mapping{
		Status: http.StatusNotFound, Code: "user_not_found",
		Message: "user not found",
	})
	Register(user.ErrRoleGroupMismatch, Mapping{
		Status: http.StatusBadRequest, Code: "role_group_mismatch",
		Message: "current_group_id is required for students and forbidden for other roles",
	})
	Register(user.ErrInvalidRole, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_role",
	})
	Register(user.ErrFullNameRequired, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_full_name",
	})
	Register(user.ErrFullNameTooLong, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_full_name",
	})

	// ------------------------------------------------------------------
	// policy
	// ------------------------------------------------------------------
	Register(policy.ErrNotFound, Mapping{
		Status: http.StatusNotFound, Code: "policy_not_found",
		Message: "policy not found",
	})
	Register(policy.ErrNoDefault, Mapping{
		Status: http.StatusBadRequest, Code: "no_default_policy",
	})
	Register(policy.ErrNameTaken, Mapping{
		Status: http.StatusConflict, Code: "policy_name_taken",
		Message: "policy name already used",
	})
	Register(policy.ErrDeletingDefault, Mapping{
		Status: http.StatusConflict, Code: "policy_default_protected",
		Message: "cannot delete default policy",
	})
	Register(policy.ErrInvalidConfig, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_config",
		// Message пуст → клиенту уйдёт err.Error() с деталями валидации
		// ("policy: invalid mechanisms config: qr_ttl.ttl_seconds must be in [3, 120]").
	})

	// ------------------------------------------------------------------
	// catalog
	// ------------------------------------------------------------------
	Register(catalog.ErrCourseNotFound, Mapping{
		Status: http.StatusNotFound, Code: "course_not_found",
		Message: "course not found",
	})
	Register(catalog.ErrCourseCodeTaken, Mapping{
		Status: http.StatusConflict, Code: "course_code_taken",
		Message: "course code already used",
	})
	Register(catalog.ErrGroupNotFound, Mapping{
		Status: http.StatusNotFound, Code: "group_not_found",
		Message: "group not found",
	})
	Register(catalog.ErrGroupNameTaken, Mapping{
		Status: http.StatusConflict, Code: "group_name_taken",
		Message: "group name already used",
	})
	Register(catalog.ErrStreamNotFound, Mapping{
		Status: http.StatusNotFound, Code: "stream_not_found",
		Message: "stream not found",
	})
	Register(catalog.ErrClassroomNotFound, Mapping{
		Status: http.StatusNotFound, Code: "classroom_not_found",
		Message: "classroom not found",
	})
	Register(catalog.ErrInUse, Mapping{
		Status: http.StatusConflict, Code: "in_use",
		Message: "entity is referenced by others",
	})

	// ------------------------------------------------------------------
	// session
	// ------------------------------------------------------------------
	Register(session.ErrNotFound, Mapping{
		Status: http.StatusNotFound, Code: "session_not_found",
		Message: "session not found",
	})
	Register(session.ErrGroupsNotInCourse, Mapping{
		Status: http.StatusConflict, Code: "groups_not_in_course_streams",
	})
	Register(session.ErrInvalidStatusTransition, Mapping{
		Status: http.StatusConflict, Code: "invalid_status_transition",
	})
	Register(session.ErrInvalidTimeRange, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_time_range",
	})
	Register(session.ErrInvalidQRTTL, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_qr_ttl",
	})
	Register(session.ErrGroupsEmpty, Mapping{
		Status: http.StatusBadRequest, Code: "groups_empty",
	})
	Register(session.ErrNotAcceptingAttendance, Mapping{
		Status: http.StatusConflict, Code: "session_not_accepting",
		Message: "session is not active or out of time range",
	})
	Register(session.ErrForbidden, Mapping{
		Status: http.StatusForbidden, Code: "forbidden",
		Message: "not authorized to operate on this session",
	})

	// ------------------------------------------------------------------
	// attendance
	// ------------------------------------------------------------------
	Register(attendance.ErrInvalidQRToken, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_qr_token",
		Message: "qr token invalid or malformed",
	})
	Register(attendance.ErrAlreadySubmitted, Mapping{
		Status: http.StatusConflict, Code: "already_submitted",
		Message: "attendance already submitted for this session",
	})
	Register(attendance.ErrNotFound, Mapping{
		Status: http.StatusNotFound, Code: "attendance_not_found",
	})
	Register(attendance.ErrInvalidFinal, Mapping{
		Status: http.StatusBadRequest, Code: "invalid_final_status",
	})
	Register(attendance.ErrNotResolvable, Mapping{
		Status: http.StatusConflict, Code: "not_resolvable",
	})
}
