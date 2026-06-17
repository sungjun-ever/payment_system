package dto

import (
	"payment_system/internal/payment"
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model

	OrderNo     string    `gorm:"type:varchar(50);not null;uniqueIndex;column:order_no"`
	UserID      uint      `gorm:"not null;index;column:user_id"`
	Status      Status    `gorm:"type:varchar(30);not null;default:PENDING;index;column:status"`
	TotalAmount uint64    `gorm:"not null;default:0;column:total_amount"`
	OrderedAt   time.Time `gorm:"not null;index;column:ordered_at"`

	OrderItems []OrderItem      `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	Payment    *payment.Payment `gorm:"foreignKey:OrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
