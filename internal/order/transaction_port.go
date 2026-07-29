package order

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	"order_system/internal/order/domain"
	productdomain "order_system/internal/product/domain"
)

type OrderUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx OrderTx) error) error
}

// OrderTx 트랜잭션 사용 모음
type OrderTx interface {
	OrderWriter() OrderWriter
	OrderReader() OrderReader
	OrderItemWriter() OrderItemWriter
	OrderItemReader() OrderItemReader
	IdempotencyWriter() IdempotencyWriter
	InventoryWriter() InventoryWriter
	InventoryJobWriter() InventoryJobWriter
}

type OrderWriter interface {
	Create(ctx context.Context, order *domain.Order) error
	CancelIfPendingByOrderID(ctx context.Context, orderID uint) (bool, error)
	CancelIfPendingByOrderAndUserID(ctx context.Context, id uint, orderNo string, userID uint) (bool, error)
}

type OrderReader interface {
	Find(ctx context.Context, id uint) (*domain.Order, error)
}

type OrderItemReader interface {
	GetItemsByOrderID(ctx context.Context, orderID uint) ([]*domain.OrderItem, error)
}

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
