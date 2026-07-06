package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/order/domain"
	"order_system/internal/pkg/apperr/dberr"

	"gorm.io/gorm"
)

var (
	ErrDuplicateOrderNo = errors.New("db: duplicate order no")
)

type OrderGormRepository struct {
	Mysql *gorm.DB
}

func NewOrderGormRepository(db *gorm.DB) OrderGormRepository {
	return OrderGormRepository{db}
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

func (r *OrderGormRepository) Find(ctx context.Context, id uint) (*domain.Order, error) {
	var order domain.Order
	result := r.Mysql.WithContext(ctx).Where("id = ?", id).First(&order)

	if result.Error != nil {
		return nil, fmt.Errorf("db: id - %c, find order by id error: %w", id, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("db: id - %c, order not found: %w", id, dberr.ErrNotFound)
	}

	return &order, nil
}

func (r *OrderGormRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	var order domain.Order
	result := r.Mysql.WithContext(ctx).Where("order_no = ?", orderNo).First(&order)

	if result.Error != nil {
		return nil, fmt.Errorf("db: order no - %s, find order by order no error: %w", orderNo, result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("db: order no - %s, order not found: %w", orderNo, dberr.ErrNotFound)
	}

	return &order, nil
}

func (r *OrderGormRepository) Update(ctx context.Context, id uint, fields map[string]interface{}) error {
	result := r.Mysql.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", id).Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("db: id - %c, update order error: %w", id, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("db: id - %c, order not found: %w", id, dberr.ErrNotFound)
	}

	return nil
}

// CancelIfPendingByOrderID 주문이 대기중 상태라면 취소 상태로 변경
func (r *OrderGormRepository) CancelIfPendingByOrderID(ctx context.Context, orderID uint) (bool, error) {
	result := r.Mysql.WithContext(ctx).
		Model(&domain.Order{}).
		Where("id = ? AND status = ?", orderID, domain.StatusPending).
		Update("status", domain.StatusCancelled)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

func (r *OrderGormRepository) CancelIfPendingByOrderAndUserID(
	ctx context.Context,
	id uint,
	orderNo string,
	userID uint,
) (bool, error) {
	result := r.Mysql.WithContext(ctx).
		Model(&domain.Order{}).
		Where("id = ? AND order_no = ? AND user_id = ? AND status = ?", id, orderNo, userID, domain.StatusPending).
		Update("status", domain.StatusCancelled)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}
