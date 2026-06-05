package apperr

import (
	"errors"
)

func ToAppError(err error) *AppError {
	switch {
	case errors.Is(err, ErrInvalidToken):
		return NewAppError(LevelWarn, 401, A002, err, nil)

	case errors.Is(err, ErrTokenNotFound):
		return NewAppError(LevelInfo, 401, A003, err, nil)

	case errors.Is(err, ErrInvalidCredentials):
		return NewAppError(LevelInfo, 401, C001, err, nil)

	case errors.Is(err, ErrTokenConflict):
		return NewAppError(LevelWarn, 409, A002, err, nil)

	case errors.Is(err, ErrResourceNotFound):
		return NewAppError(LevelInfo, 404, C002, err, nil)

	case errors.Is(err, ErrConflict):
		return NewAppError(LevelWarn, 409, C003, err, nil)

	case errors.Is(err, ErrInvalidArgument):
		return NewAppError(LevelError, 400, C001, err, nil)

	default:
		return NewAppError(LevelError, 500, S001, err, nil)
	}

}
