package errormap

import (
	"errors"
	"payment_system/internal/order/service"
	"payment_system/internal/pkg/apperr"
)

func ToAppError(err error) *apperr.AppError {
	switch {
	case errors.Is(err, service.ErrIdempotencyKeyNotFound):
		return apperr.NewAppError(apperr.LevelInfo, 409, apperr.I001, err, nil)
	case errors.Is(err, service.ErrRequestHashMismatch):
		return apperr.NewAppError(apperr.LevelWarn, 409, apperr.I002, err, nil)
	case errors.Is(err, service.ErrOrderAlreadyProcessed):
		return apperr.NewAppError(apperr.LevelInfo, 409, apperr.O002, err, nil)
	case errors.Is(err, service.ErrInsufficientProductQuantity):
		return apperr.NewAppError(apperr.LevelError, 409, apperr.O003, err, nil)
	case errors.Is(err, service.ErrDeleteOrderLockTimeout):
		return apperr.NewAppError(apperr.LevelError, 408, apperr.S002, err, nil)
	case errors.Is(err, service.ErrRestoreReservedQuantityTimeout):
		return apperr.NewAppError(apperr.LevelError, 408, apperr.S002, err, nil)
	default:
		return apperr.ToAppError(err)
	}
}
