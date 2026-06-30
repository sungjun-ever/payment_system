package domain

import (
	"time"
)

type JobStatus string
type RestoreTarget string
type RestoreOperation string

const (
	JobPending   JobStatus = "PENDING"
	JobFailed    JobStatus = "FAILED"
	JobSucceeded JobStatus = "SUCCEEDED"

	RestoreTargetDB    RestoreTarget = "DB"
	RestoreTargetRedis RestoreTarget = "REDIS"

	DecreaseReserved RestoreOperation = "DECREASE_RESERVED"
	IncreaseReserved RestoreOperation = "INCREASE_RESERVED"
)

type InventoryRestoreJob struct {
	ID          uint64           `gorm:"primaryKey;autoIncrement"`
	OrderID     uint             `gorm:"not null;uniqueIndex:uk_inventory_restore_target_order_product"`
	ProductID   uint             `gorm:"not null;uniqueIndex:uk_inventory_restore_target_order_product"`
	Target      RestoreTarget    `gorm:"type:varchar(20);not null;uniqueIndex:uk_inventory_restore_target_order_product;index:idx_inventory_restore_poll"`
	Operation   RestoreOperation `gorm:"type:varchar(50);not null"`
	OrderNo     string           `gorm:"type:varchar(50);not null;index"`
	Quantity    int              `gorm:"not null"`
	RetryCount  int              `gorm:"not null;default:0"`
	NextRetryAt time.Time        `gorm:"not null;index:idx_inventory_poll"`
	Status      JobStatus        `gorm:"type:varchar(30);not null;default:PENDING;index:idx_inventory_restore_poll"`
	LastError   string           `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
