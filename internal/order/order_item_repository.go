package order

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type OrderItemRepository interface {
	Create(ctx context.Context, tx *gorm.DB, orderItems []OrderItem) error
}

type orderItemRepository struct {
	mysql *gorm.DB
}

func NewOrderItemRepository(db *gorm.DB) OrderItemRepository {
	return &orderItemRepository{db}
}

func (o *orderItemRepository) Create(ctx context.Context, tx *gorm.DB, orderItems []OrderItem) error {
	err := gorm.G[[]OrderItem](tx).Create(ctx, &orderItems)

	if err != nil {
		return fmt.Errorf("db: create order item error: %w", err)
	}

	return nil
}
