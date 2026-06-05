package idempotency

import "payment_system/internal/model"

type CreateRequest struct {
	UserID       uint
	Scope        model.IdempotencyScope
	Key          string
	Status       model.IdempotencyStatus
	RequestHash  string
	OrderID      *uint
	PaymentID    *uint
	ResponseCode int
	Response     interface{}
}
