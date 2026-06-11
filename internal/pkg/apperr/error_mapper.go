package apperr

import (
	"errors"
	"payment_system/internal/pkg/apperr/serviceerr"
)

func ToAppError(err error) *AppError {
	switch {
	case errors.Is(err, serviceerr.ErrResourceNotFound):
		return NewAppError(LevelInfo, 404, C002, err, nil)

	case errors.Is(err, serviceerr.ErrConflict):
		return NewAppError(LevelWarn, 409, C003, err, nil)

	case errors.Is(err, serviceerr.ErrInvalidArgument):
		return NewAppError(LevelError, 400, C001, err, nil)

	default:
		return NewAppError(LevelError, 500, S001, err, nil)
	}

}
