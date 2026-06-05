package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/model"

	"gorm.io/gorm"
)

var (
	ErrDuplicateOrderNo = errors.New("duplicate order no")
)

type OrderRepository interface {
	Transaction(txFn func(tx *gorm.DB) error) error
	Create(ctx context.Context, tx *gorm.DB, order *model.Order) error
}

type orderRepository struct {
	mysql *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db}
}

func (r *orderRepository) Transaction(txFn func(tx *gorm.DB) error) error {
	return r.mysql.Transaction(txFn)
}

func (r *orderRepository) Create(ctx context.Context, tx *gorm.DB, order *model.Order) error {
	err := gorm.G[model.Order](tx).Create(ctx, order)

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("%w", ErrDuplicateOrderNo)
		}

		return fmt.Errorf("db: create order error: %w", err)
	}

	return nil
}
