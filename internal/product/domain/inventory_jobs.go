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
	Target      JobTarget
	Operation   JobOperation
	RetryCount  int
	Status      JobStatus
	Payload     string
	UniqueKey   string
	CreatedAt   time.Time
	NextRetryAt time.Time
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
	Target      JobTarget    `gorm:"type:varchar(50);not null"`
	Operation   JobOperation `gorm:"type:varchar(100);not null"`
	Status      JobStatus    `gorm:"type:varchar(30);not null;default:PENDING;index"`
	RetryCount  int          `gorm:"not null;default:0"`
	NextRetryAt time.Time    `gorm:"not null;index:idx_inventory_poll"`
	LastError   string       `gorm:"type:text"`
	Payload     string       `gorm:"type:text"`
	UniqueKey   string       `gorm:"type:varchar(255);not null;uniqueIndex:idx_inventory_unique"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
