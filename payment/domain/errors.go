package domain

import "errors"

var (
	ErrValidationFailed       = errors.New("validation failed")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)
