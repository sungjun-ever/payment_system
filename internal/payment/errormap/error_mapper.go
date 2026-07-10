package errormap

import (
	"errors"
	"order_system/internal/payment/service"
	"order_system/internal/pkg/apperr"
)

func ToAppError(err error) *apperr.AppError {
	switch {
	case errors.Is(err, service.ErrPaymentAmountMismatch):
		return apperr.NewAppError(apperr.LevelInfo, 400, apperr.P001, err, nil)
	case errors.Is(err, service.ErrOwnershipMismatch):
		return apperr.NewAppError(apperr.LevelInfo, 403, apperr.P002, err, nil)
	case errors.Is(err, service.ErrOrderNoMismatch):
		return apperr.NewAppError(apperr.LevelInfo, 400, apperr.C001, err, nil)
	case errors.Is(err, service.ErrPaymentAlreadyProcessed):
		return apperr.NewAppError(apperr.LevelInfo, 409, apperr.P003, err, nil)
	case errors.Is(err, service.ErrPaymentCompleted):
		return apperr.NewAppError(apperr.LevelInfo, 409, apperr.P004, err, nil)
	case errors.Is(err, service.ErrIdempotencyKeyNotFound):
		return apperr.NewAppError(apperr.LevelInfo, 400, apperr.I001, err, nil)
	case errors.Is(err, service.ErrRequestHashMismatch):
		return apperr.NewAppError(apperr.LevelInfo, 409, apperr.C003, err, nil)
	case errors.Is(err, service.ErrDeletePaymentLockTimeout):
		return apperr.NewAppError(apperr.LevelWarn, 408, apperr.S002, err, nil)
	case errors.Is(err, service.ErrPGRejected):
		return apperr.NewAppError(apperr.LevelInfo, 400, apperr.P005, err, nil)
	default:
		return apperr.ToAppError(err)
	}
}
