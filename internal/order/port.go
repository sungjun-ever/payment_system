package order

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	"order_system/internal/order/domain"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"
)

type OrderUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx OrderTx) error) error
}

type OrderStore interface {
	OrderUnitOfWork
	ValidateIdempotency(
		ctx context.Context,
		userID uint,
		scope idempotencydomain.Scope,
		idempotencyKey string,
		hashedRequestBody string,
	) (*idempotencydomain.IdempotencyKey, error)
	FindProduct(ctx context.Context, productID uint) (*productdomain.Product, error)
	GetOrderItems(ctx context.Context, orderID uint) ([]*domain.OrderItem, error)
}

type IdempotencyLock interface {
	GetLock(ctx context.Context, lockKey string, token string) error
	DeleteLock(ctx context.Context, lockKey string, token string) error
}

type InventoryReservation interface {
	ValidateAndUpdateReservedQuantity(ctx context.Context, keys []string, args ...interface{}) (uint, error)
	RestoreProductsReservedQuantityInRedis(
		ctx context.Context,
		orderNo string,
		orderID uint,
		items []productrepository.RestoreItem,
	) []productrepository.RestoreFailed
}

// OrderTx 트랜잭션 사용 모음
type OrderTx interface {
	OrderWriters() OrderWriter
	OrderReaders() OrderReader
	OrderItemWriters() OrderItemWriter
	IdempotencyWriters() IdempotencyWriter
	InventoryWriters() InventoryWriter
	InventoryJobWriters() InventoryJobWriter
}

// OrderWriter Order write action 모음
type OrderWriter interface {
	Create(ctx context.Context, order *domain.Order) error
	CancelIfPendingByOrderID(ctx context.Context, orderID uint) (bool, error)
	CancelIfPendingByOrderAndUserID(ctx context.Context, id uint, orderNo string, userID uint) (bool, error)
}

// OrderReader Order reader action 모음
type OrderReader interface {
	Find(ctx context.Context, id uint) (*domain.Order, error)
}

// OrderItemWriter OrderItem write action 모음
type OrderItemWriter interface {
	CreateRows(ctx context.Context, orderItems []domain.OrderItem) error
}

// IdempotencyWriter Idempotency write action 모음
type IdempotencyWriter interface {
	Update(
		ctx context.Context,
		userID uint,
		key string,
		scope idempotencydomain.Scope,
		fields map[string]interface{},
	) error
	CancelIfProcessing(
		ctx context.Context,
		orderID uint,
		userID uint,
		idempotencyKey string,
		scope idempotencydomain.Scope,
		fields map[string]interface{},
	) (bool, error)
}

type InventoryWriter interface {
	RestoreReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error
	UpdateReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error
}

type InventoryJobWriter interface {
	CreateJob(ctx context.Context, fields productdomain.InventoryJobCreateContext) error
}
