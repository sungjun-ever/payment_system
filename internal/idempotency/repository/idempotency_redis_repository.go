package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/idempotency/domain"
	"order_system/internal/pkg/apperr/rediserr"
	"order_system/internal/pkg/redisscript"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	LockTTL = 15 * time.Second
)

var (
	ErrStatusExists = errors.New("redis: idempotency status exists")
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
// rediserr.ErrLockNotOwned
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

// SetIdempotencyStatus 멱등성 상태 지정, DB 반영 실패시에 임시 저장 용도
func (r *IdempotencyRedisRepository) SetIdempotencyStatus(ctx context.Context, key string, status domain.Status) error {
	// 96시간 정도 ttl을 걸어줌
	// 개발자에게 알림을 보내 직접 DB 영속성 처리하고 삭제
	ttl := 60 * time.Minute * 72
	result, err := r.rds.SetNX(ctx, key, status, ttl).Result()

	if err != nil {
		return fmt.Errorf("redis: set idempotency status failed: %w", err)
	}

	if !result {
		return fmt.Errorf("%w", ErrStatusExists)
	}

	return nil
}

// GetIdempotencyStatus DB 영속성 실패의 경우를 생각해 확인 용도, 값이 있다면 DB 영속성과 일치하지 않는 경우
func (r *IdempotencyRedisRepository) GetIdempotencyStatus(ctx context.Context, key string) (domain.Status, error) {
	var status domain.Status
	err := r.rds.Get(ctx, key).Scan(&status)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("redis: get idempotency status failed: %w", err)
	}

	if status == "" {
		return "", nil
	}

	return status, nil
}
