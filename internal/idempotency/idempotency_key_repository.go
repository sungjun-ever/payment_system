package idempotency

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr/dberr"
	"payment_system/internal/pkg/apperr/rediserr"
	"payment_system/internal/pkg/redisscript"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrIdempotencyHashMismatch = errors.New("request hash mismatch")
)

const (
	LockTTL = 10 * time.Second
)

type IdempotencyKeyRepository interface {
	WithTx(tx *gorm.DB) IdempotencyKeyRepository
	Create(ctx context.Context, dto *IdempotencyKey) error
	Validate(
		ctx context.Context,
		userID uint,
		scope Scope,
		idempotencyKey string,
		hashedRequestBody string,
	) (*IdempotencyKey, error)
	GetLock(ctx context.Context, lockKey string, token string) error
	DeleteLock(ctx context.Context, lockKey string, token string) error
	Update(
		ctx context.Context,
		userID uint,
		key string,
		scope Scope,
		fields map[string]interface{},
	) error
}

type idempotencyKeyRepository struct {
	mysql *gorm.DB
	rds   *redis.Client
}

func (r *idempotencyKeyRepository) WithTx(tx *gorm.DB) IdempotencyKeyRepository {
	return &idempotencyKeyRepository{tx, r.rds}
}

func NewIdempotencyKeyRepository(db *gorm.DB, rds *redis.Client) IdempotencyKeyRepository {
	return &idempotencyKeyRepository{db, rds}
}

func (r *idempotencyKeyRepository) Create(ctx context.Context, idempotency *IdempotencyKey) error {
	if err := gorm.G[IdempotencyKey](r.mysql).Create(ctx, idempotency); err != nil {
		return fmt.Errorf("db: create idempotency key error: %w", err)
	}

	return nil
}

// Validate 멱등키 및 해시 요청 본문 유효성 확인 후 기존 응답 본문, 응답 코드 반환
func (r *idempotencyKeyRepository) Validate(
	ctx context.Context,
	userID uint,
	scope Scope,
	idempotencyKey string,
	hashedRequestBody string,
) (*IdempotencyKey, error) {
	key, err := gorm.G[IdempotencyKey](r.mysql).
		Where("user_id = ? AND scope = ? AND `key` = ?", userID, scope, idempotencyKey).
		First(ctx)

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

// GetLock 멱등성을 위한 락 획득
func (r *idempotencyKeyRepository) GetLock(ctx context.Context, lockKey string, token string) error {
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
func (r *idempotencyKeyRepository) DeleteLock(ctx context.Context, lockKey string, token string) error {
	result, err := redisscript.DeleteLockScript.Run(ctx, r.rds, []string{lockKey}, token).Int()

	if err != nil {
		return fmt.Errorf("redis: delete idempotency lock failed: %w", err)
	}

	if result == 0 {
		return fmt.Errorf("%w", rediserr.ErrLockNotOwned)
	}

	return nil
}

func (r *idempotencyKeyRepository) Update(
	ctx context.Context,
	userID uint,
	key string,
	scope Scope,
	fields map[string]interface{},
) error {
	result := r.mysql.WithContext(ctx).Model(&IdempotencyKey{}).
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
