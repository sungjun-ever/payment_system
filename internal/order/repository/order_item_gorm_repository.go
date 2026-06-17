package repository

import (
	"context"
	"fmt"
	"payment_system/internal/order/domain"

	"gorm.io/gorm"
)

type OrderItemGormRepository struct {
	Mysql *gorm.DB
}

func (o *OrderItemGormRepository) CreateRows(ctx context.Context, orderItems []domain.OrderItem) error {
	err := o.Mysql.WithContext(ctx).Create(&orderItems).Error

	if err != nil {
		return fmt.Errorf("db: create order item error: %w", err)
	}

	return nil
}
