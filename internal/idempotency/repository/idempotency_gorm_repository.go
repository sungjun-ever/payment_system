package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/idempotency/domain"
	"payment_system/internal/pkg/apperr/dberr"

	"gorm.io/gorm"
)

var (
	ErrIdempotencyHashMismatch = errors.New("db: request hash mismatch")
)

type IdempotencyGormRepository struct {
	mysql *gorm.DB
}

func NewIdempotencyGormRepository(db *gorm.DB) IdempotencyGormRepository {
	return IdempotencyGormRepository{db}
}

func (r IdempotencyGormRepository) WithTx(tx *gorm.DB) IdempotencyGormRepository {
	return IdempotencyGormRepository{tx}
}

func (r IdempotencyGormRepository) Create(ctx context.Context, idempotency *domain.IdempotencyKey) error {
	err := r.mysql.WithContext(ctx).Model(&idempotency.IdempotencyKey{}).Create(idempotency).Error

	if err != nil {
		return fmt.Errorf("db: create idempotency key error: %w", err)
	}

	return nil
}

// Validate 멱등키 및 해시 요청 본문 유효성 확인 후 기존 응답 본문, 응답 코드 반환
func (r IdempotencyGormRepository) Validate(
	ctx context.Context,
	userID uint,
	scope domain.Scope,
	idempotencyKey string,
	hashedRequestBody string,
) (*domain.IdempotencyKey, error) {
	var key domain.IdempotencyKey
	err := r.mysql.WithContext(ctx).
		Where("user_id = ? AND scope = ? AND `key` = ?", userID, scope, idempotencyKey).
		First(&key).
		Error

	// 저장된 멱등키가 없는 경우 오류
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w", dberr.ErrNotFound)
		}

		return nil, fmt.Errorf("db: validate idempotency key error: %w", err)
	}

	// 저장된 요청 본문이 있을 때, 현재 요청 본문과 다르면 오류
	if key.RequestHash != "" && key.RequestHash != hashedRequestBody {
		return nil, fmt.Errorf("%w", ErrIdempotencyHashMismatch)
	}

	return &key, nil
}

func (r IdempotencyGormRepository) Update(
	ctx context.Context,
	userID uint,
	key string,
	scope domain.Scope,
	fields map[string]interface{},
) error {
	result := r.mysql.WithContext(ctx).Model(&domain.IdempotencyKey{}).
		Where("user_id = ? AND key = ? AND scope = ?", userID, key, scope).
		Updates(fields)

	if result.RowsAffected == 0 {
		return fmt.Errorf("db: idempotency key %s: update idempotency key error: %w", key, dberr.ErrNotFound)
	}

	if result.Error != nil {
		return fmt.Errorf("db: idempotency key %s: update idempotency key error: %w", key, result.Error)
	}

	return nil
}
