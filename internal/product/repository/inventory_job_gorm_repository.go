package repository

import (
	"context"
	"fmt"
	"order_system/internal/product/domain"

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
