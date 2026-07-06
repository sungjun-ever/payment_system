package repository

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyrepository "order_system/internal/idempotency/repository"
	orderport "order_system/internal/order"
	"order_system/internal/order/domain"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"

	"gorm.io/gorm"
)

type orderUnitOfWork struct {
	mysql         *gorm.DB
	idempotencies idempotencyrepository.IdempotencyGormRepository
	products      productrepository.ProductGormRepository
	inventories   productrepository.InventoryGormRepository
	items         OrderItemGormRepository
}

func NewOrderUnitOfWork(
	db *gorm.DB,
	idempotencyRepo idempotencyrepository.IdempotencyGormRepository,
	productRepo productrepository.ProductGormRepository,
	inventoryRepo productrepository.InventoryGormRepository,
	itemRepo OrderItemGormRepository,
) orderport.OrderStore {
	return &orderUnitOfWork{
		mysql:         db,
		idempotencies: idempotencyRepo,
		products:      productRepo,
		inventories:   inventoryRepo,
		items:         itemRepo,
	}
}

func (u *orderUnitOfWork) ValidateIdempotency(
	ctx context.Context,
	userID uint,
	scope idempotencydomain.Scope,
	idempotencyKey string,
	hashedRequestBody string,
) (*idempotencydomain.IdempotencyKey, error) {
	return u.idempotencies.Validate(ctx, userID, scope, idempotencyKey, hashedRequestBody)
}

func (u *orderUnitOfWork) FindProduct(ctx context.Context, productID uint) (*productdomain.Product, error) {
	return u.products.Find(ctx, productID)
}

func (u *orderUnitOfWork) UpdateInventoryReservedQuantity(
	ctx context.Context,
	productID uint,
	fields map[string]interface{},
) error {
	return u.inventories.UpdateReservedQuantity(ctx, productID, fields)
}

func (u *orderUnitOfWork) GetOrderItems(ctx context.Context, orderID uint) ([]*domain.OrderItem, error) {
	return u.items.GetItemsByOrderID(ctx, orderID)
}

func (u *orderUnitOfWork) Tx(ctx context.Context, txFn func(tx orderport.OrderTx) error) error {
	return u.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&orderTx{
			orderWriter:        &OrderGormRepository{Mysql: tx},
			orderReader:        &OrderGormRepository{Mysql: tx},
			orderItemWriter:    &OrderItemGormRepository{Mysql: tx},
			idempotencyWriter:  u.idempotencies.WithTx(tx),
			inventoryWriter:    &productrepository.InventoryGormRepository{Mysql: tx},
			inventoryJobWriter: &productrepository.InventoryJobGormRepository{Mysql: tx},
		})
	})
}

type orderTx struct {
	orderWriter        orderport.OrderWriter
	orderReader        orderport.OrderReader
	orderItemWriter    orderport.OrderItemWriter
	idempotencyWriter  orderport.IdempotencyWriter
	inventoryWriter    orderport.InventoryWriter
	inventoryJobWriter orderport.InventoryJobWriter
}

func (tx *orderTx) OrderWriters() orderport.OrderWriter {
	return tx.orderWriter
}

func (tx *orderTx) OrderReaders() orderport.OrderReader { return tx.orderReader }

func (tx *orderTx) OrderItemWriters() orderport.OrderItemWriter {
	return tx.orderItemWriter
}

func (tx *orderTx) IdempotencyWriters() orderport.IdempotencyWriter {
	return tx.idempotencyWriter
}

func (tx *orderTx) InventoryWriters() orderport.InventoryWriter { return tx.inventoryWriter }

func (tx *orderTx) InventoryJobWriters() orderport.InventoryJobWriter { return tx.inventoryJobWriter }
