package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/product/domain"

	"gorm.io/gorm"
)

type InventoryGormRepository struct {
	mysql *gorm.DB
}

func NewInventoryGormRepository(db *gorm.DB) InventoryGormRepository {
	return InventoryGormRepository{db}
}

func (i *InventoryGormRepository) WithTx(tx *gorm.DB) InventoryGormRepository {
	return InventoryGormRepository{tx}
}

func (i *InventoryGormRepository) Store(ctx context.Context, inventory *domain.Inventory) error {
	result := i.mysql.WithContext(ctx).Model(&domain.Inventory{}).Create(inventory)

	if result.Error != nil {
		return fmt.Errorf("db: create inventory error: %w", result.Error)
	}

	return nil
}

func (i *InventoryGormRepository) Update(
	ctx context.Context,
	productID uint,
	fields *domain.Inventory,
) (*domain.Inventory, error) {
	result := i.mysql.WithContext(ctx).Model(&domain.Inventory{}).Where("product_id = ?", productID).Updates(fields)

	if result.Error != nil {
		return nil, fmt.Errorf("db: update inventory error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("inventory not found: %w", dberr.ErrNotFound)
	}

	inventory, err := i.FindByProductID(ctx, productID)

	if err != nil {
		return nil, err
	}

	return inventory, nil
}

func (i *InventoryGormRepository) FindByProductID(ctx context.Context, id uint) (*domain.Inventory, error) {
	var inventory domain.Inventory
	result := i.mysql.WithContext(ctx).Model(&domain.Inventory{}).Where("product_id = ?", id).First(ctx)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find inventory not found with tx error: %w", dberr.ErrNotFound)
		}

		return nil, fmt.Errorf("db: find inventory by product id error: %w", result.Error)
	}

	return &inventory, nil
}

func (i *InventoryGormRepository) UpdateReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error {
	var inventory domain.Inventory
	err := i.mysql.WithContext(ctx).Model(&inventory).Where("product_id = ?", productID).Updates(map[string]interface{}{
		"reserved_quantity": gorm.Expr("reserved_quantity + ?", fields["reserved_quantity"]),
	}).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("db: productID: %c, inventory not found: %w: %w", productID, err, dberr.ErrNotFound)
		}

		return fmt.Errorf("db: productID: %c, update inventory error: %w", productID, err)
	}

	return nil
}
