package policy

import "errors"

var (
	ErrNotFound        = errors.New("policy: not found")
	ErrNameTaken       = errors.New("policy: name already taken")
	ErrDeletingDefault = errors.New("policy: cannot delete default policy")
	ErrNoDefault       = errors.New("policy: no default policy configured")
	ErrInvalidConfig   = errors.New("policy: invalid mechanisms config")
)
