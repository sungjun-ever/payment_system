package order

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type OrderItemGormRepository struct {
	mysql *gorm.DB
}

func (o *OrderItemGormRepository) CreateRows(ctx context.Context, orderItems []OrderItem) error {
	err := o.mysql.WithContext(ctx).Create(&orderItems).Error

	if err != nil {
		return fmt.Errorf("db: create order item error: %w", err)
	}

	return nil
}
