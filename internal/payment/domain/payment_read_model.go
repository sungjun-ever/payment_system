package domain

import (
	"order_system/internal/pkg/pg"
	"time"
)

type SucceededPayment struct {
	ID                     uint
	UserID                 uint
	PaymentNo              string
	OrderNo                string
	OrderID                uint
	Status                 Payment
	PaidAt                 time.Time
	ClientIdempotencyKey   string
	Action                 AttemptAction
	Method                 Method
	AttemptStatus          AttemptStatus
	Amount                 uint64
	Provider               pg.Channel
	ProviderPaymentID      *string
	ProviderIdempotencyKey *string
}
