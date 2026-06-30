package domain

import (
	"time"
)

type JobStatus string

const (
	JobPending   JobStatus = "PENDING"
	JobFailed    JobStatus = "FAILED"
	JobSucceeded JobStatus = "SUCCEEDED"
)

type InventoryRestoreJob struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	OrderID     uint      `gorm:"not null;uniqueIndex:uk_inventory_restore_order_product"`
	ProductID   uint      `gorm:"not null;uniqueIndex:uk_inventory_restore_order_product"`
	OrderNo     string    `gorm:"type:varchar(50);not null;index"`
	Quantity    int       `gorm:"not null"`
	RetryCount  int       `gorm:"not null;default:0"`
	NextRetryAt time.Time `gorm:"not null;index:idx_inventory_poll"`
	Status      JobStatus `gorm:"type:varchar(30);not null;default:PENDING;index:idx_inventory_restore_poll"`
	LastError   string    `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
