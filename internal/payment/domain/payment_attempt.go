package domain

import (
	"order_system/internal/pkg/pg"

	"gorm.io/gorm"
)

type AttemptStatus string
type AttemptAction string

const (
	AttemptStatusPending   AttemptStatus = "PROCESSING"
	AttemptStatusSucceeded AttemptStatus = "SUCCEEDED"
	AttemptStatusFailed    AttemptStatus = "FAILED"
	AttemptStatusRejected  AttemptStatus = "REJECTED"

	AttemptActionPay    AttemptAction = "PAY"
	AttemptActionCancel AttemptAction = "CANCEL"
	AttemptActionRefund AttemptAction = "REFUND"
)

type PaymentAttempt struct {
	gorm.Model
	PaymentID uint `gorm:"not null;column:payment_id"`

	ClientIdempotencyKey string `gorm:"type:varchar(255);uniqueIndex:uk_pg;column:client_idempotency_key"`

	Action AttemptAction `gorm:"type:varchar(100);not null;column:action"`
	Method Method        `gorm:"type:varchar(100);not null;column:method"`
	Status AttemptStatus `gorm:"type:varchar(100);not null;index;column:status"`

	Amount uint64 `gorm:"not null;column:amount"`

	Provider               pg.Channel `gorm:"type:varchar(50);uniqueIndex:uk_pg;column:provider"`
	ProviderPaymentID      *string    `gorm:"type:varchar(255);uniqueIndex:uk_pg;column:provider_payment_id"`
	ProviderIdempotencyKey *string    `gorm:"type:varchar(255);uniqueIndex:uk_pg;column:provider_idempotency_key"`

	FailureReason *string `gorm:"type:varchar(255);column:failure_reason"`
}
