package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr/rediserr"
	"payment_system/internal/pkg/rediskey"
	"payment_system/internal/pkg/redisscript"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRedisInvalidQuantity      = errors.New("redis: invalid quantity")
	ErrRedisInsufficientQuantity = errors.New("redis: insufficient quantity")
)

type InventoryRedisRepository struct {
	rds *redis.Client
}

func NewInventoryRedisRepository(rds *redis.Client) InventoryRedisRepository {
	return InventoryRedisRepository{rds}
}

func (i InventoryRedisRepository) ValidateAndUpdateReservedQuantity(
	ctx context.Context,
	keys []string,
	args ...interface{},
) (uint, error) {
	result, err := redisscript.ValidateAndUpdateReservedQuantityScript.Run(
		ctx,
		i.rds,
		keys,
		args...,
	).Result()

	if err != nil {
		return 0, fmt.Errorf("redis: validate and update reserved quantity failed: %w", err)
	}

	errCode, idx := result.([]interface{})[0].(int64), result.([]interface{})[1].(int64)

	productKey := ""

	// 에러가 발생한 경우만 상품 key 가져옴
	if errCode < 1 {
		productKey = keys[int(idx)-1]
	}

	productID := uint(0)

	if productKey != "" {
		productID = uint(rediskey.ParseProductID(productKey)[0])
	}

	switch errCode {
	case 0:
		return productID, fmt.Errorf("redis: product key - %s: %w", productKey, rediserr.ErrNotFound)
	case -1:
		return productID, fmt.Errorf("redis: product key - %s: %w", productKey, ErrRedisInvalidQuantity)
	case -2:
		return productID, fmt.Errorf("redis: product key - %s: %w", productKey, ErrRedisInsufficientQuantity)
	default:
		return 0, nil
	}
}

func (i InventoryRedisRepository) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	results, err := i.rds.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis: find inventory in redis error: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("hash value is empty: %w", rediserr.ErrEmptyHash)
	}

	return results, nil
}

func (i InventoryRedisRepository) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	result, err := i.rds.HSet(ctx, key, fields).Result()

	if err != nil {
		return fmt.Errorf("store inventory in redis error: %w", err)
	}

	if result == 0 {
		return fmt.Errorf("inventory already exist: %w", rediserr.ErrConflict)
	}

	return nil
}

func (i InventoryRedisRepository) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := i.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: update inventory in redis error: %w", err)
	}

	return nil
}

func (i InventoryRedisRepository) DeleteInRedis(ctx context.Context, key string) error {
	if err := i.rds.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: delete inventory in redis error: %w", err)
	}

	return nil
}

func (i InventoryRedisRepository) UpdateReservedQuantityInRedis(ctx context.Context, keys []string, args ...interface{}) error {
	results, err := redisscript.UpdateReservedQuantitiesScript.Run(ctx, i.rds, keys, args...).Result()

	if err != nil {
		return fmt.Errorf("redis: update products reserved quantity failed: %w", err)
	}

	result, idx := results.([]interface{})[0].(int64), results.([]interface{})[1].(int64)

	if result == 0 {
		key := keys[idx-1]
		return fmt.Errorf("key: %s, update products reserved quantity failed: %w", key, rediserr.ErrNotFound)
	}

	return nil
}
