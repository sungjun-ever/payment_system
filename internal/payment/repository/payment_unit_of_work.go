package repository

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyRepository "order_system/internal/idempotency/repository"
	orderdomain "order_system/internal/order/domain"
	orderrepository "order_system/internal/order/repository"
	paymentPort "order_system/internal/payment"

	"gorm.io/gorm"
)

type paymentStore struct {
	mysql               *gorm.DB
	idempotencyGormRepo idempotencyRepository.IdempotencyGormRepository
}

func NewPaymentStore(
	db *gorm.DB,
	idempotencyRepo idempotencyRepository.IdempotencyGormRepository,
) paymentPort.PaymentStore {
	return &paymentStore{
		mysql:               db,
		idempotencyGormRepo: idempotencyRepo,
	}
}

func NewPaymentUnitOfWork(
	db *gorm.DB,
	_ PaymentAttemptGormRepository,
	idempotencyRepo idempotencyRepository.IdempotencyGormRepository,
	_ orderrepository.OrderGormRepository,
) paymentPort.PaymentUnitOfWork {
	return NewPaymentStore(db, idempotencyRepo)
}

func (p paymentStore) ValidateIdempotency(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	hashedRequestBody string,
) (*idempotencydomain.IdempotencyKey, error) {
	return p.idempotencyGormRepo.Validate(
		ctx,
		userID,
		idempotencydomain.ScopePayOrder,
		idempotencyKey,
		hashedRequestBody,
	)
}

func (p paymentStore) FindOrderForPayment(ctx context.Context, orderID uint) (*orderdomain.Order, error) {
	orderRepo := orderrepository.OrderGormRepository{Mysql: p.mysql}
	return orderRepo.Find(ctx, orderID)
}

func (p paymentStore) Tx(ctx context.Context, txFn func(tx paymentPort.PayTx) error) error {
	return p.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&paymentTx{
			paymentWriter:     &PaymentGormRepository{Mysql: tx},
			paymentReader:     &PaymentGormRepository{Mysql: tx},
			attemptWriter:     &PaymentAttemptGormRepository{Mysql: tx},
			orderWriter:       &orderrepository.OrderGormRepository{Mysql: tx},
			orderReader:       &orderrepository.OrderGormRepository{Mysql: tx},
			idempotencyWriter: p.idempotencyGormRepo.WithTx(tx),
		})
	})
}

type paymentTx struct {
	paymentWriter     paymentPort.PaymentWriter
	paymentReader     paymentPort.PaymentReader
	attemptWriter     paymentPort.AttemptWriter
	orderWriter       paymentPort.OrderWrite
	orderReader       paymentPort.OrderReader
	idempotencyWriter paymentPort.IdempotencyWrite
}

func (tx *paymentTx) PaymentsWriter() paymentPort.PaymentWriter {
	return tx.paymentWriter
}
func (tx *paymentTx) PaymentReader() paymentPort.PaymentReader  { return tx.paymentReader }
func (tx *paymentTx) AttemptsWriter() paymentPort.AttemptWriter { return tx.attemptWriter }
func (tx *paymentTx) IdempotenciesWriter() paymentPort.IdempotencyWrite {
	return tx.idempotencyWriter
}
func (tx *paymentTx) OrdersWriter() paymentPort.OrderWrite {
	return tx.orderWriter
}
func (tx *paymentTx) OrderReader() paymentPort.OrderReader { return tx.orderReader }
