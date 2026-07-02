package repository

import (
	"context"
	"fmt"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/product/domain"
	"time"

	"gorm.io/gorm"
)

type InventoryJobGormRepository struct {
	Mysql *gorm.DB
}

func NewInventoryJobGormRepository(db *gorm.DB) InventoryJobGormRepository {
	return InventoryJobGormRepository{Mysql: db}
}

func (i *InventoryJobGormRepository) CreateJob(ctx context.Context, fields domain.InventoryRestoreJobContext) error {
	result := i.Mysql.WithContext(ctx).Model(&domain.InventoryRestoreJob{}).Create(&fields)

	if result.Error != nil {
		return fmt.Errorf("db: failed to create inventory restore job: %w", result.Error)
	}

	return nil
}

func (i *InventoryJobGormRepository) UpdateJob(
	ctx context.Context,
	constraint domain.InventoryRestoreJobFindConstraint,
	fields domain.InventoryRestoreJobUpdateContext,
) error {
	result := i.Mysql.WithContext(ctx).
		Model(&domain.InventoryRestoreJob{}).
		Where(&constraint).
		Updates(&fields)

	if result.Error != nil {
		return fmt.Errorf("db: failed to update inventory restore job: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("db: no inventory restore job found with constraint: %v, %w", constraint, dberr.ErrNotFound)
	}

	return nil
}

func (i *InventoryJobGormRepository) FindDueJob(
	ctx context.Context,
	limit int,
) ([]domain.InventoryRestoreJob, error) {
	var jobs []domain.InventoryRestoreJob
	result := i.Mysql.WithContext(ctx).
		Model(&domain.InventoryRestoreJob{}).
		Where("next_retry_at <= ? AND status IN ?",
			time.Now(),
			[]domain.JobStatus{domain.JobPending, domain.JobRetryable},
		).
		Limit(limit).
		Find(&jobs)

	if result.Error != nil {
		return nil, fmt.Errorf("db: failed to find due inventory restore jobs: %w", result.Error)
	}

	return jobs, nil
}
