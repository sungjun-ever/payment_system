package order

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type OrderItemRepository interface {
	WithTx(tx *gorm.DB) OrderItemRepository
	CreateRows(ctx context.Context, orderItems []OrderItem) error
}

type orderItemRepository struct {
	mysql *gorm.DB
}

func NewOrderItemRepository(db *gorm.DB) OrderItemRepository {
	return &orderItemRepository{db}
}

func (o *orderItemRepository) WithTx(tx *gorm.DB) OrderItemRepository {
	return &orderItemRepository{tx}
}

func (o *orderItemRepository) CreateRows(ctx context.Context, orderItems []OrderItem) error {
	err := o.mysql.WithContext(ctx).Create(&orderItems).Error

	if err != nil {
		return fmt.Errorf("db: create order item error: %w", err)
	}

	return nil
}
