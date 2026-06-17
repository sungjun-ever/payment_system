package domain

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model

	PaymentNo string `gorm:"type:varchar(50);not null;uniqueIndex:uk_payments_payment_no;column:payment_no"`
	OrderID   uint64 `gorm:"not null;uniqueIndex:uk_payments_order_id;column:order_id"`

	Method Method `gorm:"type:varchar(30);not null;column:method"`
	Status Status `gorm:"type:varchar(30);not null;default:PENDING;index;column:status"`

	Amount int64 `gorm:"not null;column:amount"`

	Provider          *string `gorm:"type:varchar(50);column:provider"`
	ProviderPaymentID *string `gorm:"type:varchar(100);index;column:provider_payment_id"`
	IdempotencyKey    *string `gorm:"type:varchar(255);uniqueIndex:uk_payments_idempotency_key;column:idempotency_key"`

	FailureReason *string `gorm:"type:varchar(255);column:failure_reason"`

	PaidAt     *time.Time `gorm:"column:paid_at"`
	CanceledAt *time.Time `gorm:"column:canceled_at"`
	RefundedAt *time.Time `gorm:"column:refunded_at"`
}
