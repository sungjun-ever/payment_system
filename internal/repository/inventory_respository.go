package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/model"

	"gorm.io/gorm"
)

var (
	ErrInventoryNotFound = errors.New("db: inventory not found")
)

type InventoryRepository interface {
	Create(ctx context.Context, tx *gorm.DB, inventory *model.Inventory) error
	FindByProductID(ctx context.Context, tx *gorm.DB, id uint) (*model.Inventory, error)
}

type inventoryRepository struct {
	mysql *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db}
}

func (i inventoryRepository) Create(ctx context.Context, tx *gorm.DB, inventory *model.Inventory) error {
	err := gorm.G[model.Inventory](tx).Create(ctx, inventory)

	if err != nil {
		return fmt.Errorf("db: create inventory error: %w", err)
	}

	return nil
}

func (i inventoryRepository) FindByProductID(ctx context.Context, tx *gorm.DB, id uint) (*model.Inventory, error) {
	inventory, err := gorm.G[model.Inventory](tx).Where("product_id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrInventoryNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find inventory by product id error: %w", err)
	}

	return &inventory, nil
}
