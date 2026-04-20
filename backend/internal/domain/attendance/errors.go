package attendance

import "errors"

var (
	ErrNotFound         = errors.New("attendance: not found")
	ErrAlreadySubmitted = errors.New("attendance: already submitted for this session")
	ErrInvalidQRToken   = errors.New("attendance: invalid qr token")
	ErrInvalidFinal     = errors.New("attendance: final_status must be accepted or rejected")
	ErrNotResolvable    = errors.New("attendance: record already resolved or not eligible")
)
