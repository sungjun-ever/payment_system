package apperr

import (
	"errors"
	"order_system/internal/pkg/apperr/serviceerr"
)

func ToAppError(err error) *AppError {
	switch {
	case errors.Is(err, serviceerr.ErrResourceNotFound):
		return NewAppError(LevelInfo, 404, C002, err, nil)

	case errors.Is(err, serviceerr.ErrConflict):
		return NewAppError(LevelWarn, 409, C003, err, nil)

	case errors.Is(err, serviceerr.ErrInvalidArgument):
		return NewAppError(LevelInfo, 400, C001, err, nil)
	case errors.Is(err, serviceerr.ErrTimeout):
		return NewAppError(LevelError, 408, S002, err, nil)

	default:
		return NewAppError(LevelError, 500, S001, err, nil)
	}

}
