package serviceerr

import "errors"

var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrConflict         = errors.New("conflict")
	ErrInvalidArgument  = errors.New("invalid argument")
)
