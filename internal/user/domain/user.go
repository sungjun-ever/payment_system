package domain

import (
	Orders "payment_system/internal/order/domain"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email string `gorm:"type:varchar(100);not null;uniqueIndex;column:email"`
	Name  string `gorm:"type:varchar(50);not null;column:name"`

	Orders []Orders.Order `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
