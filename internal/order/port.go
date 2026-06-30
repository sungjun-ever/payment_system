package order

import (
	"context"
	idempotencyDomain "order_system/internal/idempotency/domain"
	"order_system/internal/order/domain"
	productDomain "order_system/internal/product/domain"
)

type OrderUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx OrderTx) error) error
}

type OrderStore interface {
	OrderUnitOfWork
	ValidateIdempotency(
		ctx context.Context,
		userID uint,
		scope idempotencyDomain.Scope,
		idempotencyKey string,
		hashedRequestBody string,
	) (*idempotencyDomain.IdempotencyKey, error)
	FindProduct(ctx context.Context, productID uint) (*productDomain.Product, error)
	GetOrderItems(ctx context.Context, orderID uint) ([]*domain.OrderItem, error)
}

type IdempotencyLock interface {
	GetLock(ctx context.Context, lockKey string, token string) error
	DeleteLock(ctx context.Context, lockKey string, token string) error
}

type InventoryReservation interface {
	ValidateAndUpdateReservedQuantity(ctx context.Context, keys []string, args ...interface{}) (uint, error)
	UpdateReservedQuantityInRedis(ctx context.Context, keys []string, args ...interface{}) error
}

// OrderTx 트랜잭션 사용 모음
type OrderTx interface {
	OrderWriters() OrderWriter
	OrderReaders() OrderReader
	OrderItemWriters() OrderItemWriter
	IdempotencyWriters() IdempotencyWriter
	InventoryWriters() InventoryWriter
}

// OrderWriter Order write action 모음
type OrderWriter interface {
	Create(ctx context.Context, order *domain.Order) error
	CancelIfPendingByOrderNo(ctx context.Context, orderNo string) (bool, error)
	CancelIfPendingByOrderIDAndUserID(ctx context.Context, id uint, userID uint) (bool, error)
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
		scope idempotencyDomain.Scope,
		fields map[string]interface{},
	) error
	CancelIfProcessingByOrderIDAndUserID(
		ctx context.Context,
		orderID uint,
		userID uint,
	) (bool, error)
	CancelIfProcessingByOrderNoAndUserID(ctx context.Context, orderNo string, userID uint) (bool, error)
}

type InventoryWriter interface {
	RestoreReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error
	UpdateReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error
}
