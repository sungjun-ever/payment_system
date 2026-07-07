package domain

import (
	"time"

	"gorm.io/gorm"
)

type PaymentStatus string

const (
	Processing PaymentStatus = "PROCESSING"
	Succeeded  PaymentStatus = "SUCCEEDED"
	Rejected   PaymentStatus = "REJECTED"
	Failed     PaymentStatus = "FAILED"
)

type Payment struct {
	gorm.Model

	UserID    uint          `gorm:"not null;index;column:user_id"`
	PaymentNo string        `gorm:"type:varchar(50);not null;uniqueIndex:uk_payments_payment_no;column:payment_no"`
	OrderID   uint          `gorm:"not null;uniqueIndex:uk_payments_order_id;column:order_id"`
	Status    PaymentStatus `gorm:"type:varchar(30);not null;index;column:status"`

	PaidAt     *time.Time `gorm:"column:paid_at"`
	CanceledAt *time.Time `gorm:"column:canceled_at"`
	RefundedAt *time.Time `gorm:"column:refunded_at"`
}
