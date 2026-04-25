package auth

import "errors"

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
	ErrInvalidEmail        = errors.New("email is invalid")
	ErrInvalidPassword     = errors.New("password must be at least 8 characters")
)
