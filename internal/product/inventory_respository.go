package product

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr/dberr"
	"payment_system/internal/pkg/apperr/rediserr"
	"payment_system/internal/pkg/rediskey"
	"payment_system/internal/pkg/redisscript"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrRedisInvalidQuantity      = errors.New("redis: invalid quantity")
	ErrRedisInsufficientQuantity = errors.New("redis: insufficient quantity")
)

type InventoryRepository interface {
	WithTx(tx *gorm.DB) InventoryRepository
	Store(ctx context.Context, inventory *Inventory) error
	Update(ctx context.Context, productID uint, fields *Inventory) (*Inventory, error)
	FindByProductID(ctx context.Context, id uint) (*Inventory, error)
	ValidateAndUpdateReservedQuantity(ctx context.Context, keys []string, args ...interface{}) (uint, error)
	GetInventoryLock(ctx context.Context, lockKey string, token string) error
	DeleteInventoryLock(ctx context.Context, lockKey string, token string) error
	FindInRedis(ctx context.Context, key string) (map[string]string, error)
	StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	DeleteInRedis(ctx context.Context, key string) error
	UpdateReservedQuantityInRedis(ctx context.Context, keys []string, args ...interface{}) error
	UpdateReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error
}

type inventoryRepository struct {
	mysql *gorm.DB
	rds   *redis.Client
}

func NewInventoryRepository(db *gorm.DB, rds *redis.Client) InventoryRepository {
	return &inventoryRepository{db, rds}
}

func (i inventoryRepository) WithTx(tx *gorm.DB) InventoryRepository {
	return &inventoryRepository{tx, i.rds}
}

func (i inventoryRepository) Store(ctx context.Context, inventory *Inventory) error {
	result := i.mysql.WithContext(ctx).Model(&Inventory{}).Create(inventory)

	if result.Error != nil {
		return fmt.Errorf("db: create inventory error: %w", result.Error)
	}

	return nil
}

func (i inventoryRepository) Update(
	ctx context.Context,
	productID uint,
	fields *Inventory,
) (*Inventory, error) {
	result := i.mysql.WithContext(ctx).Model(&Inventory{}).Where("product_id = ?", productID).Updates(fields)

	if result.Error != nil {
		return nil, fmt.Errorf("db: update inventory error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("inventory not found: %w", dberr.ErrNotFound)
	}

	inventory, err := i.FindByProductID(ctx, productID)

	if err != nil {
		return nil, err
	}

	return inventory, nil
}

func (i inventoryRepository) FindByProductID(ctx context.Context, id uint) (*Inventory, error) {
	var inventory Inventory
	result := i.mysql.WithContext(ctx).Model(&Inventory{}).Where("product_id = ?", id).First(ctx)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find inventory not found with tx error: %w", dberr.ErrNotFound)
		}

		return nil, fmt.Errorf("db: find inventory by product id error: %w", result.Error)
	}

	return &inventory, nil
}

func (i inventoryRepository) GetInventoryLock(ctx context.Context, lockKey string, token string) error {
	result, err := i.rds.SetNX(ctx, lockKey, token, rediskey.InventoryLockTTL).Result()

	if err != nil {
		return fmt.Errorf("redis: get inventory lock failed: %w", err)
	}

	if !result {
		return fmt.Errorf("cant get inventory lock: %w", rediserr.ErrLockExists)
	}

	return nil
}

func (i inventoryRepository) DeleteInventoryLock(ctx context.Context, lockKey string, token string) error {
	result, err := redisscript.DeleteLockScript.Run(ctx, i.rds, []string{lockKey}, token).Int()

	if err != nil {
		return fmt.Errorf("redis: delete inventory lock failed: %w", err)
	}

	switch result {
	case 0:
		return fmt.Errorf("lock owner already exist: %w", rediserr.ErrLockNotOwned)
	default:
		return nil
	}
}

func (i inventoryRepository) ValidateAndUpdateReservedQuantity(
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

func (i inventoryRepository) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	results, err := i.rds.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis: find inventory in redis error: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("hash value is empty: %w", rediserr.ErrEmptyHash)
	}

	return results, nil
}

func (i inventoryRepository) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	result, err := i.rds.HSet(ctx, key, fields).Result()

	if err != nil {
		return fmt.Errorf("store inventory in redis error: %w", err)
	}

	if result == 0 {
		return fmt.Errorf("inventory already exist: %w", rediserr.ErrConflict)
	}

	return nil
}

func (i inventoryRepository) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := i.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: update inventory in redis error: %w", err)
	}

	return nil
}

func (i inventoryRepository) DeleteInRedis(ctx context.Context, key string) error {
	if err := i.rds.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: delete inventory in redis error: %w", err)
	}

	return nil
}

func (i inventoryRepository) UpdateReservedQuantityInRedis(ctx context.Context, keys []string, args ...interface{}) error {
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

func (i inventoryRepository) UpdateReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error {
	var inventory Inventory
	err := i.mysql.WithContext(ctx).Model(&inventory).Where("product_id = ?", productID).Updates(map[string]interface{}{
		"reserved_quantity": gorm.Expr("reserved_quantity + ?", fields["reserved_quantity"]),
	}).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("db: productID: %c, inventory not found: %w: %w", productID, err, dberr.ErrNotFound)
		}

		return fmt.Errorf("db: productID: %c, update inventory error: %w", productID, err)
	}

	return nil
}
