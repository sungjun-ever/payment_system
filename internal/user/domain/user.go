package domain

import (
	"order_system/internal/order/domain"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email    string `gorm:"type:varchar(100);not null;uniqueIndex;column:email"`
	Name     string `gorm:"type:varchar(50);not null;column:name"`
	Password string `gorm:"type:varchar(255);not null;column:password"`

	Orders []domain.Order `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
