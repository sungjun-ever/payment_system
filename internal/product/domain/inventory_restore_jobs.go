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
	JobRetryable JobStatus = "RETRYABLE"
	JobSucceeded JobStatus = "SUCCEEDED"

	RestoreTargetDB    RestoreTarget = "DB"
	RestoreTargetRedis RestoreTarget = "REDIS"

	DecreaseReserved RestoreOperation = "DECREASE_RESERVED"
	IncreaseReserved RestoreOperation = "INCREASE_RESERVED"
)

type InventoryRestoreJobContext struct {
	OrderNo     string
	ProductID   uint
	Target      RestoreTarget
	Operation   RestoreOperation
	Quantity    int
	RetryCount  int
	Status      JobStatus
	CreatedAt   time.Time
	NextRetryAt time.Time
}

type InventoryRestoreJobFindConstraint struct {
	OrderNo   string
	ProductID uint
	Target    RestoreTarget
	Operation RestoreOperation
}

type InventoryRestoreJobUpdateContext struct {
	RetryCount  int
	Status      JobStatus
	NextRetryAt time.Time
	LastError   string
	UpdatedAt   time.Time
}

type InventoryRestoreJob struct {
	ID          uint64           `gorm:"primaryKey;autoIncrement"`
	OrderNo     string           `gorm:"type:varchar(50);not null;uniqueIndex:uk_inventory_restore_job"`
	ProductID   uint             `gorm:"not null;uniqueIndex:uk_inventory_restore_job"`
	Target      RestoreTarget    `gorm:"type:varchar(20);not null;uniqueIndex:uk_inventory_restore_job"`
	Operation   RestoreOperation `gorm:"type:varchar(50);not null"`
	Quantity    int              `gorm:"not null"`
	RetryCount  int              `gorm:"not null;default:0"`
	NextRetryAt time.Time        `gorm:"not null;index:idx_inventory_poll"`
	Status      JobStatus        `gorm:"type:varchar(30);not null;default:PENDING;index"`
	LastError   string           `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
