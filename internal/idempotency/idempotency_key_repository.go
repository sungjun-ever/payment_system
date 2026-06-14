package idempotency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type IdempotencyKeyRepository interface {
	Create(ctx context.Context, dto *IdempotencyKey) error
	Get(
		ctx context.Context,
		userID uint,
		scope Scope,
		idempotencyKey string,
	) (*IdempotencyKey, error)
}

type idempotencyKeyRepository struct {
	mysql *gorm.DB
}

func NewIdempotencyKeyRepository(db *gorm.DB) IdempotencyKeyRepository {
	return &idempotencyKeyRepository{db}
}

func (r *idempotencyKeyRepository) Create(ctx context.Context, idempotency *IdempotencyKey) error {
	if err := gorm.G[IdempotencyKey](r.mysql).Create(ctx, idempotency); err != nil {
		return fmt.Errorf("db: create idempotency key error: %w", err)
	}

	return nil
}

func (r *idempotencyKeyRepository) Get(
	ctx context.Context,
	userID uint,
	scope Scope,
	idempotencyKey string,
) (*IdempotencyKey, error) {
	key, err := gorm.G[IdempotencyKey](r.mysql).
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
