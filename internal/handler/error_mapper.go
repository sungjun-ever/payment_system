package handler

import (
	"errors"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/service"
)

func toAppError(err error) *apperr.AppError {
	switch {
	case errors.Is(err, service.ErrInvalidToken):
		return apperr.NewAppError(apperr.LevelWarn, 401, apperr.A002, err, nil)

	case errors.Is(err, service.ErrTokenNotFound):
		return apperr.NewAppError(apperr.LevelInfo, 401, apperr.A003, err, nil)

	case errors.Is(err, service.ErrInvalidCredentials):
		return apperr.NewAppError(apperr.LevelInfo, 401, apperr.C001, err, nil)

	case errors.Is(err, service.ErrTokenConflict):
		return apperr.NewAppError(apperr.LevelWarn, 409, apperr.A002, err, nil)

	case errors.Is(err, service.ErrResourceNotFound):
		return apperr.NewAppError(apperr.LevelInfo, 404, apperr.C002, err, nil)

	case errors.Is(err, service.ErrConflict):
		return apperr.NewAppError(apperr.LevelWarn, 409, apperr.C003, err, nil)

	case errors.Is(err, service.ErrInvalidArgument):
		return apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil)

	default:
		return apperr.NewAppError(apperr.LevelError, 500, apperr.S001, err, nil)
	}

}
