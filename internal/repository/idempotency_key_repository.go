package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/model"

	"gorm.io/gorm"
)

type IdempotencyKeyRepository interface {
	Create(ctx context.Context, tx *gorm.DB, dto *model.IdempotencyKey) error
	Get(
		ctx context.Context,
		userID uint,
		scope model.IdempotencyScope,
		idempotencyKey string,
	) (*model.IdempotencyKey, error)
}

type idempotencyKeyRepository struct {
	mysql *gorm.DB
}

func NewIdempotencyKeyRepository(db *gorm.DB) IdempotencyKeyRepository {
	return &idempotencyKeyRepository{db}
}

func (r *idempotencyKeyRepository) Create(ctx context.Context, tx *gorm.DB, dto *model.IdempotencyKey) error {
	if tx == nil {
		tx = r.mysql
	}

	if err := gorm.G[model.IdempotencyKey](tx).Create(ctx, dto); err != nil {
		return fmt.Errorf("db: create idempotency key error: %w", err)
	}

	return nil
}

func (r *idempotencyKeyRepository) Get(
	ctx context.Context,
	userID uint,
	scope model.IdempotencyScope,
	idempotencyKey string,
) (*model.IdempotencyKey, error) {
	key, err := gorm.G[model.IdempotencyKey](r.mysql).
		Where("user_id = ? AND scope = ? AND `key` = ?", userID, scope, idempotencyKey).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("db: get idempotency key error: %w", err)
	}

	return &key, nil
}
