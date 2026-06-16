package serviceerr

import "errors"

var (
	ErrResourceNotFound   = errors.New("resource not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrNoPermissionAtLock = errors.New("no permission at lock")
	ErrTimeout            = errors.New("timeout")
)
