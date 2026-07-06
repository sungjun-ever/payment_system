package product

import (
	"context"
	productrepository "order_system/internal/product/repository"

	"gorm.io/gorm"
)

type productStore struct {
	mysql *gorm.DB
}

func NewProductStore(mysql *gorm.DB) ProductStore {
	return &productStore{
		mysql: mysql,
	}
}

func (p productStore) Tx(ctx context.Context, txFn func(tx ProductTx) error) error {
	return p.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&productTx{
			inventoryWriter:         &productrepository.InventoryGormRepository{Mysql: tx},
			inventoryMovementWriter: &productrepository.InventoryMovementGormRepository{Mysql: tx},
		})
	})
}

type productTx struct {
	inventoryWriter         InventoryWriter
	inventoryMovementWriter InventoryMovementWriter
}

func (tx *productTx) InventoryWriter() InventoryWriter {
	return tx.inventoryWriter
}

func (tx *productTx) InventoryMovementWriter() InventoryMovementWriter {
	return tx.inventoryMovementWriter
}
