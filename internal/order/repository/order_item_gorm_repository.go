package repository

import (
	"context"
	"fmt"
	"order_system/internal/order/domain"
	"order_system/internal/pkg/apperr/dberr"

	"gorm.io/gorm"
)

type OrderItemGormRepository struct {
	Mysql *gorm.DB
}

func NewOrderItemGormRepository(db *gorm.DB) OrderItemGormRepository {
	return OrderItemGormRepository{db}
}

func (o *OrderItemGormRepository) CreateRows(ctx context.Context, orderItems []domain.OrderItem) error {
	err := o.Mysql.WithContext(ctx).Create(&orderItems).Error

	if err != nil {
		return fmt.Errorf("db: create order item error: %w", err)
	}

	return nil
}

func (o *OrderItemGormRepository) GetItemsByOrderID(ctx context.Context, orderID uint) ([]*domain.OrderItem, error) {
	var orderItems []*domain.OrderItem
	result := o.Mysql.WithContext(ctx).Where("order_id = ?", orderID).Find(&orderItems)

	if result.Error != nil {
		return nil, fmt.Errorf("db: get order items by order id error: %w", result.Error)
	}

	return orderItems, nil
}

func (o *OrderItemGormRepository) GetItemByOrderIDAndProductID(
	ctx context.Context,
	orderID uint,
	productID uint,
) (*domain.OrderItem, error) {
	var orderItem domain.OrderItem
	result := o.Mysql.WithContext(ctx).Where("order_id = ? AND product_id = ?", orderID, productID).First(&orderItem)

	if result.Error != nil {
		return nil, fmt.Errorf("db: get order item by order id and product id error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("db: get order item by order id and product id error: %w", dberr.ErrNotFound)
	}

	return &orderItem, nil
}
