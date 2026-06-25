package repository

import (
	"context"
	idempotencyDomain "order_system/internal/idempotency/domain"
	idempotencyRepository "order_system/internal/idempotency/repository"
	orderPort "order_system/internal/order"
	productDomain "order_system/internal/product/domain"
	productRepository "order_system/internal/product/repository"

	"gorm.io/gorm"
)

type orderUnitOfWork struct {
	mysql         *gorm.DB
	idempotencies idempotencyRepository.IdempotencyGormRepository
	products      productRepository.ProductGormRepository
	inventories   productRepository.InventoryGormRepository
}

func NewOrderUnitOfWork(
	db *gorm.DB,
	idempotencyRepo idempotencyRepository.IdempotencyGormRepository,
	productRepo productRepository.ProductGormRepository,
	inventoryRepo productRepository.InventoryGormRepository,
) orderPort.OrderStore {
	return &orderUnitOfWork{
		mysql:         db,
		idempotencies: idempotencyRepo,
		products:      productRepo,
		inventories:   inventoryRepo,
	}
}

func (u *orderUnitOfWork) ValidateIdempotency(
	ctx context.Context,
	userID uint,
	scope idempotencyDomain.Scope,
	idempotencyKey string,
	hashedRequestBody string,
) (*idempotencyDomain.IdempotencyKey, error) {
	return u.idempotencies.Validate(ctx, userID, scope, idempotencyKey, hashedRequestBody)
}

func (u *orderUnitOfWork) FindProduct(ctx context.Context, productID uint) (*productDomain.Product, error) {
	return u.products.Find(ctx, productID)
}

func (u *orderUnitOfWork) UpdateInventoryReservedQuantity(
	ctx context.Context,
	productID uint,
	fields map[string]interface{},
) error {
	return u.inventories.UpdateReservedQuantity(ctx, productID, fields)
}

func (u *orderUnitOfWork) Tx(ctx context.Context, txFn func(tx orderPort.OrderTx) error) error {
	return u.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&orderTx{
			orderWriter:       &OrderGormRepository{Mysql: tx},
			orderItemWriter:   &OrderItemGormRepository{Mysql: tx},
			idempotencyWriter: u.idempotencies.WithTx(tx),
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
