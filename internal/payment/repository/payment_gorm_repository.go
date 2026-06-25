package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/payment/domain"
	"order_system/internal/pkg/apperr/dberr"

	"gorm.io/gorm"
)

type PaymentGormRepository struct {
	Mysql *gorm.DB
}

func NewPaymentGormRepository(db *gorm.DB) PaymentGormRepository {
	return PaymentGormRepository{db}
}

func (p *PaymentGormRepository) Create(ctx context.Context, payment *domain.Payment) (*domain.Payment, error) {
	err := p.Mysql.WithContext(ctx).Model(&domain.Payment{}).Create(payment).Error

	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (p *PaymentGormRepository) FindByUserAndOrderID(ctx context.Context, userID, orderID uint) (*domain.Payment, error) {
	var payment domain.Payment
	result := p.Mysql.WithContext(ctx).
		Where("user_id = ? AND order_id = ?", userID, orderID).
		Find(&payment)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: find payment by user and order id error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, nil
	}

	return &payment, nil
}

func (p *PaymentGormRepository) Update(ctx context.Context, paymentID uint, fields map[string]interface{}) error {
	result := p.Mysql.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("id = ?", paymentID).
		Updates(map[string]interface{}{
			"status":  fields["status"],
			"paid_at": fields["paid_at"],
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("id: %c, payment not found: %w", paymentID, dberr.ErrNotFound)
	}

	return nil
}
