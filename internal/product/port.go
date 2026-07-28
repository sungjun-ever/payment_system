package product

import (
	"context"
	"order_system/internal/product/domain"

	"gorm.io/gorm"
)

type ProductRepository interface {
	Transaction(txFn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) ProductRepository
	Store(ctx context.Context, product *domain.Product) error
	Find(ctx context.Context, id uint) (*domain.Product, error)
	Update(ctx context.Context, id uint, fields *domain.Product) (*domain.Product, error)
	Delete(ctx context.Context, id uint) error
}

type ProductCache interface {
	FindInRedis(ctx context.Context, key string) (map[string]string, error)
	StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	DeleteInRedis(ctx context.Context, key string) error
}

type InventoryRepository interface {
	WithTx(tx *gorm.DB) InventoryRepository
	Store(ctx context.Context, inventory *domain.Inventory) error
	FindByProductID(ctx context.Context, id uint) (*domain.Inventory, error)
	Update(ctx context.Context, id uint, fields *domain.Inventory) (*domain.Inventory, error)
}

type InventoryCache interface {
	FindInRedis(ctx context.Context, key string) (map[string]string, error)
	StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	DeleteInRedis(ctx context.Context, key string) error
}
