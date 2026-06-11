package rediserr

import "errors"

var (
	ErrNotFound  = errors.New("redis: value not found")
	ErrEmptyHash = errors.New("redis: hash empty")
	ErrConflict  = errors.New("redis: conflict")

	ErrLockExists   = errors.New("redis: lock exists")
	ErrLockNotOwned = errors.New("redis: lock not owned")
)
