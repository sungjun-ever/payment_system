package payment

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	orderdomain "order_system/internal/order/domain"
	"order_system/internal/payment/domain"
	productdomain "order_system/internal/product/domain"
)

type PaymentIdempotencyReader interface {
	Validate(
		ctx context.Context,
		userID uint,
		scope idempotencydomain.Scope,
		idempotencyKey string,
		hashedRequestBody string,
	) (*idempotencydomain.IdempotencyKey, error)
}

type PaymentIdempotencyStatusStore interface {
	GetIdempotencyStatus(ctx context.Context, key string) (idempotencydomain.Status, error)
	SetIdempotencyStatus(ctx context.Context, key string, status idempotencydomain.Status) error
}

type PaymentOrderReader interface {
	Find(ctx context.Context, id uint) (*orderdomain.Order, error)
}

type PaymentOrderItemReader interface {
	GetItemsByOrderID(ctx context.Context, orderID uint) ([]*orderdomain.OrderItem, error)
}

type PaymentInventoryJobWriter interface {
	CreateJob(ctx context.Context, fields productdomain.InventoryJobCreateContext) error
}

type PaymentIdempotencyGuard interface {
	GetLock(ctx context.Context, lockKey string, token string) error
	DeleteLock(ctx context.Context, lockKey string, token string) error
}

type PaymentReader interface {
	FindByUserAndOrderID(ctx context.Context, userID, orderID uint) (*domain.Payment, error)
	Find(ctx context.Context, paymentID uint) (*domain.Payment, error)
	FindPaymentAndSucceededAttempt(ctx context.Context, paymentID uint) (*domain.SucceededPayment, error)
}
