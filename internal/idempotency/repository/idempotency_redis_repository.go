package repository

import (
	"context"
	"fmt"
	"payment_system/internal/pkg/apperr/rediserr"
	"payment_system/internal/pkg/redisscript"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	LockTTL = 10 * time.Second
)

type IdempotencyRedisRepository struct {
	rds *redis.Client
}

func NewIdempotencyRedisRepository(rds *redis.Client) IdempotencyRedisRepository {
	return IdempotencyRedisRepository{rds}
}

// GetLock 멱등성을 위한 락 획득
func (r *IdempotencyRedisRepository) GetLock(ctx context.Context, lockKey string, token string) error {
	result, err := r.rds.SetNX(ctx, lockKey, token, LockTTL).Result()

	if err != nil {
		return fmt.Errorf("redis: get idempotency lock failed: %w", err)
	}

	if !result {
		return fmt.Errorf("%w", rediserr.ErrLockExists)
	}

	return nil
}

// DeleteLock 락 삭제, 권한 없으면 오류
func (r *IdempotencyRedisRepository) DeleteLock(ctx context.Context, lockKey string, token string) error {
	result, err := redisscript.DeleteLockScript.Run(ctx, r.rds, []string{lockKey}, token).Int()

	if err != nil {
		return fmt.Errorf("redis: delete idempotency lock failed: %w", err)
	}

	if result == 0 {
		return fmt.Errorf("%w", rediserr.ErrLockNotOwned)
	}

	return nil
}
