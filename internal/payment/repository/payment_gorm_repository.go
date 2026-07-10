package repository

import (
	"context"
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

func (p *PaymentGormRepository) Find(ctx context.Context, id uint) (*domain.Payment, error) {
	var payment domain.Payment
	result := p.Mysql.WithContext(ctx).Where("id = ?", id).Find(&payment)

	if result.Error != nil {
		return nil, fmt.Errorf("db: find payment by id error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("payment not found: %w", dberr.ErrNotFound)
	}

	return &payment, nil
}

func (p *PaymentGormRepository) FindByUserAndOrderID(ctx context.Context, userID, orderID uint) (*domain.Payment, error) {
	var payment domain.Payment
	result := p.Mysql.WithContext(ctx).
		Where("user_id = ? AND order_id = ?", userID, orderID).
		Find(&payment)

	if result.Error != nil {
		return nil, fmt.Errorf("db: find payment by user and order id error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, nil
	}

	return &payment, nil
}

func (p *PaymentGormRepository) UpdatePaidStatus(ctx context.Context, paymentID uint, fields map[string]interface{}) error {
	result := p.Mysql.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("id = ?", paymentID).
		Updates(fields)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("id: %d, payment not found: %w", paymentID, dberr.ErrNotFound)
	}

	return nil
}

func (p *PaymentGormRepository) UpdateRefundStatus(ctx context.Context, paymentID uint, fields map[string]interface{}) error {
	result := p.Mysql.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("id = ?", paymentID).
		Updates(fields)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("id: %d, payment not found: %w", paymentID, dberr.ErrNotFound)
	}

	return nil
}

func (p *PaymentGormRepository) FindPaymentAndSucceededAttempt(
	ctx context.Context,
	paymentID uint,
) (*domain.SucceededPayment, error) {
	var succeededPayment domain.SucceededPayment
	result := p.Mysql.WithContext(ctx).Model(&domain.Payment{}).
		Select("payments.*", "orders.order_no", "payment_attempts.client_idempotency_key", "payment_attempts.action",
			"payment_attempts.method", "payment_attempts.status as attempt_status", "payment_attempts.amount",
			"payment_attempts.provider", "payment_attempts.provider_payment_id", "payment_attempts.provider_idempotency_key").
		Joins("JOIN orders ON orders.id = payments.order_id").
		Joins("JOIN payment_attempts ON payment_attempts.payment_id = payments.id").
		Where("payments.id = ? AND payments.refunded_at IS NULL AND payment_attempts.action = ? AND payment_attempts.status = ?",
			paymentID, domain.AttemptActionPay, domain.AttemptStatusSucceeded).
		Find(&succeededPayment)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("id: %d, succeeded payment not found: %w", paymentID, dberr.ErrNotFound)
	}

	return &succeededPayment, nil
}
