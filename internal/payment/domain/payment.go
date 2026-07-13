package domain

import (
	"time"

	"gorm.io/gorm"
)

type PaymentStatus string
type RefundStatus string

const (
	PaidProcessing PaymentStatus = "PROCESSING"
	PaidSucceeded  PaymentStatus = "SUCCEEDED"
	PaidRejected   PaymentStatus = "REJECTED"
	PaidFailed     PaymentStatus = "FAILED"

	RefundProcessing PaymentStatus = "PROCESSING"
	RefundSucceeded  PaymentStatus = "SUCCEEDED"
	RefundRejected   PaymentStatus = "REJECTED"
	RefundFailed     PaymentStatus = "FAILED"
)

type Payment struct {
	gorm.Model

	UserID    uint          `gorm:"not null;index;column:user_id"`
	PaymentNo string        `gorm:"type:varchar(50);not null;uniqueIndex:uk_payments_payment_no;column:payment_no"`
	OrderID   uint          `gorm:"not null;uniqueIndex:uk_payments_order_id;column:order_id"`
	Status    PaymentStatus `gorm:"type:varchar(30);not null;index;column:status"`

	PaidAt      *time.Time `gorm:"column:paid_at"`
	CancelledAt *time.Time `gorm:"column:cancelled_at"`
	RefundedAt  *time.Time `gorm:"column:refunded_at"`
}
