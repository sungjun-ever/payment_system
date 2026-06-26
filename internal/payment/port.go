package payment

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	orderdomain "order_system/internal/order/domain"
	"order_system/internal/payment/domain"
)

type PaymentUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx PayTx) error) error
}

type PaymentStore interface {
	PaymentUnitOfWork
	ValidateIdempotency(
		ctx context.Context,
		userID uint,
		idempotencyKey string,
		hashedRequestBody string,
	) (*idempotencydomain.IdempotencyKey, error)
	FindOrderForPayment(ctx context.Context, orderID uint) (*orderdomain.Order, error)
}

type IdempotencyGuard interface {
	GetLock(ctx context.Context, lockKey string, token string) error
	DeleteLock(ctx context.Context, lockKey string, token string) error
	SetIdempotencyStatus(ctx context.Context, key string, status idempotencydomain.Status) error
	GetIdempotencyStatus(ctx context.Context, key string) (idempotencydomain.Status, error)
}

type PayTx interface {
	PaymentsWriter() PaymentWriter
	PaymentReader() PaymentReader
	AttemptsWriter() AttemptWriter
	AttemptReader() AttemptReader
	IdempotenciesWriter() IdempotencyWrite
	IdempotenciesReader() IdempotencyReader
	OrdersWriter() OrderWrite
	OrderReader() OrderReader
}

type PaymentWriter interface {
	Create(ctx context.Context, payment *domain.Payment) (*domain.Payment, error)
	Update(ctx context.Context, paymentID uint, fields map[string]interface{}) error
}

type PaymentReader interface {
	FindByUserAndOrderID(ctx context.Context, userID, orderID uint) (*domain.Payment, error)
	Find(ctx context.Context, paymentID uint) (*domain.Payment, error)
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
