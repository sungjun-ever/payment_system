package order

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrDuplicateOrderNo = errors.New("db: duplicate order no")
)

type OrderRepository interface {
	WithTx(tx *gorm.DB) OrderRepository
	Transaction(txFn func(tx *gorm.DB) error) error
	Create(ctx context.Context, order *Order) error
}

type orderRepository struct {
	mysql *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db}
}

func (r *orderRepository) WithTx(tx *gorm.DB) OrderRepository {
	return &orderRepository{tx}
}

func (r *orderRepository) Transaction(txFn func(tx *gorm.DB) error) error {
	return r.mysql.Transaction(txFn)
}

func (r *orderRepository) Create(ctx context.Context, order *Order) error {
	err := r.mysql.WithContext(ctx).Create(order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("%w", ErrDuplicateOrderNo)
		}

		return fmt.Errorf("db: create order error: %w", err)
	}

	return nil
}
