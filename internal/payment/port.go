package payment

import (
	"context"
	idempotencyDomain "order_system/internal/idempotency/domain"
	orderdomain "order_system/internal/order/domain"
	"order_system/internal/payment/domain"
)

type PaymentUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx PayTx) error) error
}

type PayTx interface {
	PaymentsWriter() PaymentWriter
	PaymentReader() PaymentReader
	AttemptsWriter() AttemptWriter
	IdempotenciesWriter() IdempotencyWrite
	OrdersWriter() OrderWrite
	OrderReader() OrderReader
}

type PaymentWriter interface {
	Create(ctx context.Context, payment *domain.Payment) (*domain.Payment, error)
	Update(ctx context.Context, paymentID uint, fields map[string]interface{}) error
}

type PaymentReader interface {
	FindByUserAndOrderID(ctx context.Context, userID, orderID uint) (*domain.Payment, error)
}

type AttemptWriter interface {
	Create(ctx context.Context, attempt *domain.PaymentAttempt) (*domain.PaymentAttempt, error)
	Update(ctx context.Context, attemptID uint, fields map[string]interface{}) error
}

type IdempotencyWrite interface {
	Update(
		ctx context.Context,
		userID uint,
		key string,
		scope idempotencyDomain.Scope,
		fields map[string]interface{},
	) error
}

type OrderWrite interface {
	Update(ctx context.Context, id uint, fields map[string]interface{}) error
}

type OrderReader interface {
	Find(ctx context.Context, id uint) (*orderdomain.Order, error)
}
