package session

import "errors"

var (
	ErrNotFound                = errors.New("session: not found")
	ErrInvalidStatus           = errors.New("session: invalid status")
	ErrInvalidStatusTransition = errors.New("session: invalid status transition")
	ErrInvalidTimeRange        = errors.New("session: ends_at must be after starts_at")
	ErrInvalidQRTTL            = errors.New("session: qr_ttl_seconds out of range")
	ErrQRSecretLen             = errors.New("session: qr_secret must be 32 bytes")
	ErrGroupsEmpty             = errors.New("session: at least one group is required")
	ErrGroupsNotInCourse       = errors.New("session: selected groups do not belong to course streams")
	ErrNotAcceptingAttendance  = errors.New("session: not accepting attendance")
	ErrForbidden               = errors.New("session: not authorized to operate on this session")
)
