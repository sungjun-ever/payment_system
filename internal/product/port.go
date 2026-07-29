package product

import (
	"context"
	"order_system/internal/product/domain"
)

type ProductUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx ProductTx) error) error
}

type ProductTx interface {
	ProductStore() ProductStore
	InventoryStore() InventoryStore
}

type ProductStore interface {
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

type InventoryStore interface {
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
