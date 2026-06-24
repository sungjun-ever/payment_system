package repository

import (
	"context"
	idempotencyRepository "order_system/internal/idempotency/repository"
	orderrepository "order_system/internal/order/repository"
	paymentPort "order_system/internal/payment"

	"gorm.io/gorm"
)

type paymentUnitOfWork struct {
	mysql                  *gorm.DB
	paymentAttemptGormRepo PaymentAttemptGormRepository
	idempotencyGormRepo    idempotencyRepository.IdempotencyGormRepository
	orderGormRepo          orderrepository.OrderGormRepository
}

func NewPaymentUnitOfWork(
	db *gorm.DB,
	paymentAttemptGormRepo PaymentAttemptGormRepository,
	idempotencyRepo idempotencyRepository.IdempotencyGormRepository,
	orderGormRepo orderrepository.OrderGormRepository,
) paymentPort.PaymentUnitOfWork {
	return &paymentUnitOfWork{
		db,
		paymentAttemptGormRepo,
		idempotencyRepo,
		orderGormRepo,
	}
}

func (p paymentUnitOfWork) Tx(ctx context.Context, txFn func(tx paymentPort.PayTx) error) error {
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
