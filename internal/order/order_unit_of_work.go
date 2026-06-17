package order

import (
	"context"
	"payment_system/internal/idempotency"
	"payment_system/internal/order/repository"

	"gorm.io/gorm"
)

type orderUnitOfWork struct {
	mysql           *gorm.DB
	idempotencyRepo idempotency.IdempotencyGormRepository
}

func NewOrderUnitOfWork(
	db *gorm.DB,
	idempotencyRepo idempotency.IdempotencyGormRepository,
) OrderUnitOfWork {
	return &orderUnitOfWork{
		mysql:           db,
		idempotencyRepo: idempotencyRepo,
	}
}

func (u *orderUnitOfWork) Tx(ctx context.Context, txFn func(tx OrderTx) error) error {
	return u.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&orderTx{
			orderWriter:       &repository.OrderGormRepository{Mysql: tx},
			orderItemWriter:   &repository.OrderItemGormRepository{Mysql: tx},
			idempotencyWriter: u.idempotencyRepo.WithTx(tx),
		})
	})
}

type orderTx struct {
	orderWriter       OrderWriter
	orderItemWriter   OrderItemWriter
	idempotencyWriter IdempotencyWriter
}

func (tx *orderTx) Orders() OrderWriter {
	return tx.orderWriter
}

func (tx *orderTx) OrderItems() OrderItemWriter {
	return tx.orderItemWriter
}

func (tx *orderTx) Idempotencies() IdempotencyWriter {
	return tx.idempotencyWriter
}
