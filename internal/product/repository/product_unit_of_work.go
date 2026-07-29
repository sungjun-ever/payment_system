package repository

import (
	"context"
	productport "order_system/internal/product"

	"gorm.io/gorm"
)

type productUnitOfWork struct {
	mysql *gorm.DB
}

func NewProductUnitOfWork(db *gorm.DB) productport.ProductUnitOfWork {
	return &productUnitOfWork{
		mysql: db,
	}
}

func (u *productUnitOfWork) Tx(ctx context.Context, txFn func(tx productport.ProductTx) error) error {
	return u.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&productTx{
			productStore:   &ProductGormRepository{Mysql: tx},
			inventoryStore: &InventoryGormRepository{Mysql: tx},
		})
	})
}

type productTx struct {
	productStore   productport.ProductStore
	inventoryStore productport.InventoryStore
}

func (tx *productTx) ProductStore() productport.ProductStore {
	return tx.productStore
}

func (tx *productTx) InventoryStore() productport.InventoryStore {
	return tx.inventoryStore
}
