package audit

import "errors"

var (
	ErrChainBroken    = errors.New("audit: hash chain is broken")
	ErrInvalidHashLen = errors.New("audit: hash must be 32 bytes")
	ErrInvalidPayload = errors.New("audit: payload is not canonicalizable")
)
