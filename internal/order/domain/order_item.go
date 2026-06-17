package domain

import (
	"gorm.io/gorm"
)

type OrderItem struct {
	gorm.Model

	OrderID   uint `gorm:"not null;index;column:order_id"`
	ProductID uint `gorm:"not null;index;column:product_id"`

	ProductName string `gorm:"type:varchar(100);not null;column:product_name"`
	UnitPrice   uint64 `gorm:"not null;column:unit_price"`
	Quantity    int    `gorm:"not null;column:quantity"`
	TotalPrice  uint64 `gorm:"not null;column:total_price"`
}
