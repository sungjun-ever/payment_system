package dberr

import "errors"

var (
	ErrNotFound  = errors.New("db: entity not found")
	ErrDuplicate = errors.New("db: duplicate entity")
)
