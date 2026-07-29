package payment

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	orderdomain "order_system/internal/order/domain"
	"order_system/internal/payment/domain"
	productdomain "order_system/internal/product/domain"
)

type PaymentUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx PayTx) error) error
}

type PayTx interface {
	PaymentsWriter() PaymentWriter
	PaymentsReader() PaymentReader
	AttemptsWriter() AttemptWriter
	AttemptsReader() AttemptReader
	IdempotenciesWriter() IdempotencyWrite
	IdempotenciesReader() IdempotencyReader
	OrdersWriter() OrderWrite
	OrdersReader() OrderReader
	InventoryWriter() InventoryWriter
	InventoryMovementWriter() InventoryMovementWriter
}

type PaymentWriter interface {
	Create(ctx context.Context, payment *domain.Payment) (*domain.Payment, error)
	UpdatePaidStatus(ctx context.Context, paymentID uint, fields map[string]interface{}) error
}

type AttemptWriter interface {
	Create(ctx context.Context, attempt *domain.PaymentAttempt) (*domain.PaymentAttempt, error)
	Update(ctx context.Context, attemptID uint, fields map[string]interface{}) error
}

type AttemptReader interface {
	Find(ctx context.Context, attemptID uint) (*domain.PaymentAttempt, error)
}

type IdempotencyWrite interface {
	Update(
		ctx context.Context,
		userID uint,
		key string,
		scope idempotencydomain.Scope,
		fields map[string]interface{},
	) error
}

type IdempotencyReader interface {
	FindByConstraint(
		ctx context.Context,
		userID uint,
		scope idempotencydomain.Scope,
		key string,
	) (*idempotencydomain.IdempotencyKey, error)
}

type OrderWrite interface {
	Update(ctx context.Context, id uint, fields map[string]interface{}) error
}

type OrderReader interface {
	Find(ctx context.Context, id uint) (*orderdomain.Order, error)
}

type InventoryWriter interface {
	IncreaseSoldAndDecreaseReservedQuantity(ctx context.Context, productID uint, quantity int) error
	DecreaseSoldQuantity(ctx context.Context, productID uint, quantity int) error
}

type InventoryMovementWriter interface {
	CreateInventoryMovement(ctx context.Context, entity *productdomain.InventoryMovement) error
}
