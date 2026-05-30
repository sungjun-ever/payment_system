package domain

import (
	"gorm.io/gorm"
)

type OrderItem struct {
	gorm.Model

	OrderID   uint64 `gorm:"not null;index;column:order_id"`
	ProductID uint64 `gorm:"not null;index;column:product_id"`

	ProductName string `gorm:"type:varchar(100);not null;column:product_name"`
	UnitPrice   int64  `gorm:"not null;column:unit_price"`
	Quantity    int    `gorm:"not null;column:quantity"`
	TotalPrice  int64  `gorm:"not null;column:total_price"`
}
