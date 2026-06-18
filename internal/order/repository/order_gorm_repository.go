package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/order/domain"

	"gorm.io/gorm"
)

var (
	ErrDuplicateOrderNo = errors.New("db: duplicate order no")
)

type OrderGormRepository struct {
	Mysql *gorm.DB
}

func (r *OrderGormRepository) Create(ctx context.Context, order *domain.Order) error {
	err := r.Mysql.WithContext(ctx).Create(order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("%w", ErrDuplicateOrderNo)
		}

		return fmt.Errorf("db: create order error: %w", err)
	}

	return nil
}
