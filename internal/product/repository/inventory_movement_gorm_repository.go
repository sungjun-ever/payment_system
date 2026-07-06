package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/product/domain"

	"gorm.io/gorm"
)

type InventoryMovementGormRepository struct {
	Mysql *gorm.DB
}

func NewInventoryMovementGormRepository(mysql *gorm.DB) InventoryMovementGormRepository {
	return InventoryMovementGormRepository{
		Mysql: mysql,
	}
}

func (r *InventoryMovementGormRepository) CreateInventoryMovement(ctx context.Context, entity *domain.InventoryMovement) error {
	err := r.Mysql.WithContext(ctx).Create(entity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("entity: %v, inventory movement already exists: %w",
				entity, dberr.ErrDuplicate)
		}

		return fmt.Errorf("failed to create inventory movement: %w", err)
	}

	return nil
}
