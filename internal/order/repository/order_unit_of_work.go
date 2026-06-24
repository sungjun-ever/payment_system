package repository

import (
	"context"
	idempotencyRepository "order_system/internal/idempotency/repository"
	orderPort "order_system/internal/order"

	"gorm.io/gorm"
)

type orderUnitOfWork struct {
	mysql           *gorm.DB
	idempotencyRepo idempotencyRepository.IdempotencyGormRepository
}

func NewOrderUnitOfWork(
	db *gorm.DB,
	idempotencyRepo idempotencyRepository.IdempotencyGormRepository,
) orderPort.OrderUnitOfWork {
	return &orderUnitOfWork{
		mysql:           db,
		idempotencyRepo: idempotencyRepo,
	}
}

func (u *orderUnitOfWork) Tx(ctx context.Context, txFn func(tx orderPort.OrderTx) error) error {
	return u.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&orderTx{
			orderWriter:       &OrderGormRepository{Mysql: tx},
			orderItemWriter:   &OrderItemGormRepository{Mysql: tx},
			idempotencyWriter: u.idempotencyRepo.WithTx(tx),
		})
	})
}

type orderTx struct {
	orderWriter       orderPort.OrderWriter
	orderItemWriter   orderPort.OrderItemWriter
	idempotencyWriter orderPort.IdempotencyWriter
}

func (tx *orderTx) Orders() orderPort.OrderWriter {
	return tx.orderWriter
}

func (tx *orderTx) OrderItems() orderPort.OrderItemWriter {
	return tx.orderItemWriter
}

func (tx *orderTx) Idempotencies() orderPort.IdempotencyWriter {
	return tx.idempotencyWriter
}
