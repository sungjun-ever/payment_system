package apperr

import "errors"

var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenConflict      = errors.New("token conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")

	ErrResourceNotFound = errors.New("resource not found")
	ErrConflict         = errors.New("conflict")
	ErrInvalidArgument  = errors.New("invalid argument")
)
