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

func (i *InventoryJobGormRepository) CreateJob(ctx context.Context, fields domain.InventoryJobCreateContext) error {
	result := i.Mysql.WithContext(ctx).Model(&domain.InventoryJob{}).Create(&fields)

	if result.Error != nil {
		return fmt.Errorf("db: failed to create inventory restore job: %w", result.Error)
	}

	return nil
}

func (i *InventoryJobGormRepository) UpdateJob(
	ctx context.Context,
	constraint domain.InventoryJobFindConstraint,
	fields domain.InventoryJobUpdateContext,
) error {
	result := i.Mysql.WithContext(ctx).
		Model(&domain.InventoryJob{}).
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
) ([]domain.InventoryJob, error) {
	var jobs []domain.InventoryJob
	result := i.Mysql.WithContext(ctx).
		Model(&domain.InventoryJob{}).
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
