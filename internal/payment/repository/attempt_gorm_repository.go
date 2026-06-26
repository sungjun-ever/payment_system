package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/payment/domain"
	"order_system/internal/pkg/apperr/dberr"

	"gorm.io/gorm"
)

type PaymentAttemptGormRepository struct {
	Mysql *gorm.DB
}

func NewAttemptGormRepository(db *gorm.DB) PaymentAttemptGormRepository {
	return PaymentAttemptGormRepository{db}
}

func (a *PaymentAttemptGormRepository) Create(
	ctx context.Context,
	attempt *domain.PaymentAttempt,
) (*domain.PaymentAttempt, error) {
	err := a.Mysql.WithContext(ctx).Model(&domain.PaymentAttempt{}).Create(attempt).Error

	if err != nil {
		return nil, err
	}

	return attempt, nil
}

func (a *PaymentAttemptGormRepository) Find(ctx context.Context, id uint) (*domain.PaymentAttempt, error) {
	var attempt domain.PaymentAttempt
	result := a.Mysql.WithContext(ctx).Where("id = ?", id).First(&attempt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("payment attempt not found: %w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: id - %c, find payment attempt by id error: %w", id, result.Error)
	}
	return &attempt, nil
}

func (a *PaymentAttemptGormRepository) Update(ctx context.Context, attemptID uint, fields map[string]interface{}) error {
	result := a.Mysql.WithContext(ctx).Model(&domain.PaymentAttempt{}).Where("id = ?", attemptID).Updates(fields)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("attempt id: %c, payment attempt not found: %w", attemptID, dberr.ErrNotFound)
	}

	return nil
}
