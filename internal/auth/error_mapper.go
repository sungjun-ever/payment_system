package auth

import (
	"errors"
	"payment_system/internal/pkg/apperr"
)

func ToAppError(err error) *apperr.AppError {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return apperr.NewAppError(apperr.LevelWarn, 401, apperr.A005, err, nil)
	case errors.Is(err, ErrInvalidToken):
		return apperr.NewAppError(apperr.LevelWarn, 401, apperr.A002, err, nil)
	case errors.Is(err, ErrTokenAlreadyExist):
		return apperr.NewAppError(apperr.LevelWarn, 409, apperr.A004, err, nil)
	default:
		return apperr.ToAppError(err)
	}
}
