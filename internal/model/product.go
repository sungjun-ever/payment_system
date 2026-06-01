package model

import (
	"gorm.io/gorm"
)

type Product struct {
	gorm.Model
	Name        string        `gorm:"type:varchar(100);not null;column:name"`
	Description *string       `gorm:"type:varchar(1000);column:description"`
	Price       int64         `gorm:"not null;column:price"`
	Status      ProductStatus `gorm:"type:varchar(30);not null;default:ACTIVE;index;column:status"`

	Inventory  *Inventory  `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`
	OrderItems []OrderItem `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`
}
