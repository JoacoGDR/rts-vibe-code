package auth

import "errors"

var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenInvalid       = errors.New("token invalid or expired")
	ErrTicketInvalid      = errors.New("ws ticket invalid or expired")
)
