package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/pkg/apperr/rediserr"
	"order_system/internal/pkg/rediskey"
	"order_system/internal/pkg/redisscript"
	"order_system/internal/product/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRedisNoneReservedQuantity = errors.New("redis: none reserved quantity")
	ErrRedisInvalidQuantity      = errors.New("redis: invalid quantity")
	ErrRedisInsufficientQuantity = errors.New("redis: insufficient quantity")
)

type RestoreItem struct {
	ProductID uint
	Quantity  int
}

type RestoreFailed struct {
	OrderNo   string
	ProductID uint
	Quantity  int
	Operation domain.RestoreOperation
	Err       error
}

type InventoryRedisRepository struct {
	rds *redis.Client
}

func NewInventoryRedisRepository(rds *redis.Client) InventoryRedisRepository {
	return InventoryRedisRepository{rds}
}

func (i *InventoryRedisRepository) ValidateAndUpdateReservedQuantity(
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

func (i *InventoryRedisRepository) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	results, err := i.rds.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis: find inventory in redis error: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("hash value is empty: %w", rediserr.ErrEmptyHash)
	}

	return results, nil
}

func (i *InventoryRedisRepository) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	result, err := i.rds.HSet(ctx, key, fields).Result()

	if err != nil {
		return fmt.Errorf("store inventory in redis error: %w", err)
	}

	if result == 0 {
		return fmt.Errorf("inventory already exist: %w", rediserr.ErrConflict)
	}

	return nil
}

func (i *InventoryRedisRepository) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := i.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: update inventory in redis error: %w", err)
	}

	return nil
}

func (i *InventoryRedisRepository) DeleteInRedis(ctx context.Context, key string) error {
	if err := i.rds.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: delete inventory in redis error: %w", err)
	}

	return nil
}

func (i *InventoryRedisRepository) RestoreProductsReservedQuantityInRedis(
	ctx context.Context,
	orderNo string,
	items []RestoreItem,
) []RestoreFailed {
	keys, args := i.setRestoreScriptKeysAndArgs(orderNo, items)

	results, err := redisscript.RestoreReservedQuantitiesScript.Run(ctx, i.rds, keys, args...).Result()

	if err != nil {
		var fails []RestoreFailed
		for _, item := range items {
			fails = append(fails, RestoreFailed{
				OrderNo:   orderNo,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Operation: domain.DecreaseReserved,
				Err: fmt.Errorf("redis: product %d restore reserved quantity failed: %w",
					item.ProductID, err),
			})
		}
		return fails
	}

	fails := make([]RestoreFailed, 0)
	for _, result := range results.([]interface{}) {
		res, idx := result.([]interface{})[0].(int64), result.([]interface{})[1].(int64)
		item := items[int(idx)-1]
		restoreResult := i.handleRestoreResult(res, orderNo, item)

		if restoreResult != nil {
			fails = append(fails, *restoreResult)
		}
	}

	return fails
}

func (i *InventoryRedisRepository) RestoreProductReservedQuantityInRedis(
	ctx context.Context,
	orderNo string,
	item RestoreItem,
) *RestoreFailed {
	keys, args := i.setRestoreScriptKeysAndArgs(orderNo, []RestoreItem{item})
	results, err := redisscript.RestoreReservedQuantityScript.
		Run(ctx, i.rds, keys, args...).
		Result()

	if err != nil {
		fail := &RestoreFailed{
			OrderNo:   orderNo,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Operation: domain.DecreaseReserved,
			Err: fmt.Errorf("redis: product %v, restore reserved quantity failed: %w",
				item, err),
		}
		return fail
	}

	result := results.(int64)
	return i.handleRestoreResult(result, orderNo, item)
}

func (i *InventoryRedisRepository) setRestoreScriptKeysAndArgs(
	orderNo string,
	items []RestoreItem,
) (keys []string, args []interface{}) {
	for _, item := range items {
		keys = append(keys, rediskey.ProductInventoryKey(item.ProductID))
		args = append(args, item.Quantity)
	}

	for _, item := range items {
		keys = append(keys, rediskey.InventoryRestoreDoneKey(orderNo, item.ProductID, string(domain.DecreaseReserved)))
	}

	args = append(args, int64((72 * time.Hour).Seconds()))

	return keys, args
}

func (i *InventoryRedisRepository) handleRestoreResult(
	restoreResult int64,
	orderNo string,
	item RestoreItem,
) *RestoreFailed {
	switch restoreResult {
	case 0: // 예약 재고 존재 없음
		return &RestoreFailed{
			OrderNo:   orderNo,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Operation: domain.DecreaseReserved,
			Err: fmt.Errorf("redis: product %v, orderNo %s, reserved quantity not exist: %w",
				item, orderNo, ErrRedisNoneReservedQuantity),
		}
	case -1: // 입력값 오류
		return &RestoreFailed{
			OrderNo:   orderNo,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Operation: domain.DecreaseReserved,
			Err: fmt.Errorf("redis: product %v, orderNo %s, restore reserved quantity failed: %w",
				item, orderNo, ErrRedisInvalidQuantity),
		}
	}

	return nil
}
