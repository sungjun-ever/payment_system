package product

import (
	"gorm.io/gorm"
)

type Inventory struct {
	gorm.Model
	ProductID        uint `gorm:"not null;uniqueIndex;column:product_id"`
	TotalQuantity    int  `gorm:"not null;default:0;column:total_quantity"`
	ReservedQuantity int  `gorm:"not null;default:0;column:reserved_quantity"`
	SoldQuantity     int  `gorm:"not null;default:0;column:sold_quantity"`
}
