package order

import (
	"context"
	idempotencyDomain "order_system/internal/idempotency/domain"
	"order_system/internal/order/domain"
)

type OrderUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx OrderTx) error) error
}

// OrderTx 트랜잭션 사용 모음
type OrderTx interface {
	Orders() OrderWriter
	OrderItems() OrderItemWriter
	Idempotencies() IdempotencyWriter
}

// OrderWriter Order write action 모음
type OrderWriter interface {
	Create(ctx context.Context, order *domain.Order) error
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
}
