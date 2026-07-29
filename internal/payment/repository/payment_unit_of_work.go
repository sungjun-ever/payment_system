package repository

import (
	"context"
	idempotencyrepository "order_system/internal/idempotency/repository"
	orderrepository "order_system/internal/order/repository"
	paymentport "order_system/internal/payment"
	productrepository "order_system/internal/product/repository"

	"gorm.io/gorm"
)

type paymentUnitOfWork struct {
	mysql *gorm.DB
}

func NewPaymentUnitOfWork(
	db *gorm.DB,
) paymentport.PaymentUnitOfWork {
	return &paymentUnitOfWork{
		mysql: db,
	}
}

func (p *paymentUnitOfWork) Tx(ctx context.Context, txFn func(tx paymentport.PayTx) error) error {
	return p.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&paymentTx{
			paymentWriter:           &PaymentGormRepository{Mysql: tx},
			paymentReader:           &PaymentGormRepository{Mysql: tx},
			attemptWriter:           &PaymentAttemptGormRepository{Mysql: tx},
			attemptReader:           &PaymentAttemptGormRepository{Mysql: tx},
			orderWriter:             &orderrepository.OrderGormRepository{Mysql: tx},
			orderReader:             &orderrepository.OrderGormRepository{Mysql: tx},
			idempotencyWriter:       &idempotencyrepository.IdempotencyGormRepository{Mysql: tx},
			idempotencyReader:       &idempotencyrepository.IdempotencyGormRepository{Mysql: tx},
			inventoryWriter:         &productrepository.InventoryGormRepository{Mysql: tx},
			inventoryMovementWriter: &productrepository.InventoryMovementGormRepository{Mysql: tx},
		})
	})
}

type paymentTx struct {
	paymentWriter           paymentport.PaymentWriter
	paymentReader           paymentport.PaymentReader
	attemptWriter           paymentport.AttemptWriter
	attemptReader           paymentport.AttemptReader
	orderWriter             paymentport.OrderWrite
	orderReader             paymentport.OrderReader
	idempotencyWriter       paymentport.IdempotencyWrite
	idempotencyReader       paymentport.IdempotencyReader
	inventoryWriter         paymentport.InventoryWriter
	inventoryMovementWriter paymentport.InventoryMovementWriter
}

func (tx *paymentTx) PaymentsWriter() paymentport.PaymentWriter {
	return tx.paymentWriter
}
func (tx *paymentTx) PaymentsReader() paymentport.PaymentReader { return tx.paymentReader }
func (tx *paymentTx) AttemptsWriter() paymentport.AttemptWriter { return tx.attemptWriter }
func (tx *paymentTx) AttemptsReader() paymentport.AttemptReader { return tx.attemptReader }
func (tx *paymentTx) IdempotenciesWriter() paymentport.IdempotencyWrite {
	return tx.idempotencyWriter
}
func (tx *paymentTx) IdempotenciesReader() paymentport.IdempotencyReader {
	return tx.idempotencyReader
}
func (tx *paymentTx) OrdersWriter() paymentport.OrderWrite {
	return tx.orderWriter
}
func (tx *paymentTx) OrdersReader() paymentport.OrderReader        { return tx.orderReader }
func (tx *paymentTx) InventoryWriter() paymentport.InventoryWriter { return tx.inventoryWriter }
func (tx *paymentTx) InventoryMovementWriter() paymentport.InventoryMovementWriter {
	return tx.inventoryMovementWriter
}
