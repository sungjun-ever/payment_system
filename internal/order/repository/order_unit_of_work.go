package repository

import (
	"context"
	idempotencyrepository "order_system/internal/idempotency/repository"
	orderport "order_system/internal/order"
	productrepository "order_system/internal/product/repository"

	"gorm.io/gorm"
)

type orderUnitOfWork struct {
	mysql *gorm.DB
}

func NewOrderUnitOfWork(
	db *gorm.DB,
) orderport.OrderUnitOfWork {
	return &orderUnitOfWork{
		mysql: db,
	}
}

func (u *orderUnitOfWork) Tx(ctx context.Context, txFn func(tx orderport.OrderTx) error) error {
	return u.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&orderTx{
			orderWriter:        &OrderGormRepository{Mysql: tx},
			orderReader:        &OrderGormRepository{Mysql: tx},
			orderItemWriter:    &OrderItemGormRepository{Mysql: tx},
			orderItemReader:    &OrderItemGormRepository{Mysql: tx},
			idempotencyWriter:  &idempotencyrepository.IdempotencyGormRepository{Mysql: tx},
			inventoryWriter:    &productrepository.InventoryGormRepository{Mysql: tx},
			inventoryJobWriter: &productrepository.InventoryJobGormRepository{Mysql: tx},
		})
	})
}

type orderTx struct {
	orderWriter        orderport.OrderWriter
	orderReader        orderport.OrderReader
	orderItemWriter    orderport.OrderItemWriter
	orderItemReader    orderport.OrderItemReader
	idempotencyWriter  orderport.IdempotencyWriter
	inventoryWriter    orderport.InventoryWriter
	inventoryJobWriter orderport.InventoryJobWriter
}

func (tx *orderTx) OrderWriter() orderport.OrderWriter {
	return tx.orderWriter
}

func (tx *orderTx) OrderReader() orderport.OrderReader { return tx.orderReader }

func (tx *orderTx) OrderItemWriter() orderport.OrderItemWriter {
	return tx.orderItemWriter
}

func (tx *orderTx) OrderItemReader() orderport.OrderItemReader {
	return tx.orderItemReader
}

func (tx *orderTx) IdempotencyWriter() orderport.IdempotencyWriter {
	return tx.idempotencyWriter
}

func (tx *orderTx) InventoryWriter() orderport.InventoryWriter { return tx.inventoryWriter }

func (tx *orderTx) InventoryJobWriter() orderport.InventoryJobWriter { return tx.inventoryJobWriter }
