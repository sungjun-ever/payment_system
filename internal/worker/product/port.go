package product

import (
	"context"
	productdomain "order_system/internal/product/domain"
)

type ProductUnitOfWork interface {
	Tx(ctx context.Context, txFn func(tx ProductTx) error) error
}

type ProductStore interface {
	ProductUnitOfWork
}

type ProductTx interface {
	InventoryWriter() InventoryWriter
	InventoryMovementWriter() InventoryMovementWriter
}

type InventoryWriter interface {
	UpdateSoldQuantity(ctx context.Context, productID uint, quantity int) error
}

type InventoryMovementWriter interface {
	CreateInventoryMovement(ctx context.Context, entity *productdomain.InventoryMovement) error
}
