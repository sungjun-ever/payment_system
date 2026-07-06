package domain

import "gorm.io/gorm"

type InventoryMovement struct {
	gorm.Model
	OrderID   uint         `gorm:"not null;uniqueKey:idx_movement"`
	ProductID uint         `gorm:"not null;uniqueKey:idx_movement"`
	Operation JobOperation `gorm:"not null;uniqueKey:idx_movement"`
	Quantity  int          `gorm:"not null"`
}
