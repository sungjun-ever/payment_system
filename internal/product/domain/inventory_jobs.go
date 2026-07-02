package domain

import (
	"time"
)

type JobStatus string
type JobTarget string
type JobOperation string

const (
	JobPending   JobStatus = "PENDING"
	JobFailed    JobStatus = "FAILED"
	JobRetryable JobStatus = "RETRYABLE"
	JobSucceeded JobStatus = "SUCCEEDED"

	TargetDB    JobTarget = "DB"
	TargetRedis JobTarget = "REDIS"

	DecreaseReserved JobOperation = "DECREASE_RESERVED"
	IncreaseReserved JobOperation = "INCREASE_RESERVED"
	ConfirmSale      JobOperation = "CONFIRM_SALE"
)

type InventoryJobCreateContext struct {
	OrderNo     string
	ProductID   uint
	Target      JobTarget
	Operation   JobOperation
	Quantity    int
	RetryCount  int
	Status      JobStatus
	CreatedAt   time.Time
	NextRetryAt time.Time
}

type InventoryJobFindConstraint struct {
	OrderNo   string
	ProductID uint
	Target    JobTarget
	Operation JobOperation
}

type InventoryJobUpdateContext struct {
	RetryCount  int
	Status      JobStatus
	NextRetryAt time.Time
	LastError   string
	UpdatedAt   time.Time
}

type InventoryJob struct {
	ID          uint64       `gorm:"primaryKey;autoIncrement"`
	OrderNo     string       `gorm:"type:varchar(50);not null;uniqueIndex:uk_inventory_restore_job"`
	ProductID   uint         `gorm:"not null;uniqueIndex:uk_inventory_restore_job"`
	Target      JobTarget    `gorm:"type:varchar(20);not null;uniqueIndex:uk_inventory_restore_job"`
	Operation   JobOperation `gorm:"type:varchar(50);not null"`
	Quantity    int          `gorm:"not null"`
	RetryCount  int          `gorm:"not null;default:0"`
	NextRetryAt time.Time    `gorm:"not null;index:idx_inventory_poll"`
	Status      JobStatus    `gorm:"type:varchar(30);not null;default:PENDING;index"`
	LastError   string       `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
