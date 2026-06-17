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

type OrderGormRepository struct {
	mysql *gorm.DB
}

func (r *OrderGormRepository) Create(ctx context.Context, order *Order) error {
	err := r.mysql.WithContext(ctx).Create(order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("%w", ErrDuplicateOrderNo)
		}

		return fmt.Errorf("db: create order error: %w", err)
	}

	return nil
}
