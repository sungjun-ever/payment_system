package product

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/redisscript"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrInventoryNotFound           = errors.New("db: inventory not found")
	ErrRedisInventoryNotFound      = errors.New("redis: inventory not found")
	ErrRedisInvalidQuantity        = errors.New("redis: invalid quantity")
	ErrRedisInsufficientQuantity   = errors.New("redis: insufficient quantity")
	ErrRedisInventoryAlreadyExists = errors.New("redis: inventory already exists")
)

type InventoryRepository interface {
	FindByProductID(ctx context.Context, id uint) (*Inventory, error)
	CreateWithTransaction(ctx context.Context, tx *gorm.DB, inventory *Inventory) error
	FindByProductIDWithTransaction(ctx context.Context, tx *gorm.DB, id uint) (*Inventory, error)
	ValidateAndUpdateReservedQuantity(ctx context.Context, keys []string, args ...interface{}) error
	GetInventoryLock(ctx context.Context, lockKey string, token string) (bool, error)
	SetInventory(ctx context.Context, key string, fields map[string]interface{}) error
}

type inventoryRepository struct {
	mysql *gorm.DB
	rds   *redis.Client
}

func NewInventoryRepository(db *gorm.DB, rds *redis.Client) InventoryRepository {
	return &inventoryRepository{db, rds}
}

func (i inventoryRepository) FindByProductID(ctx context.Context, id uint) (*Inventory, error) {
	inventory, err := gorm.G[Inventory](i.mysql).Where("product_id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrInventoryNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find inventory by product id error: %w", err)
	}

	return &inventory, nil
}

func (i inventoryRepository) CreateWithTransaction(ctx context.Context, tx *gorm.DB, inventory *Inventory) error {
	err := gorm.G[Inventory](tx).Create(ctx, inventory)

	if err != nil {
		return fmt.Errorf("db: create inventory error: %w", err)
	}

	return nil
}

func (i inventoryRepository) FindByProductIDWithTransaction(ctx context.Context, tx *gorm.DB, id uint) (*Inventory, error) {
	inventory, err := gorm.G[Inventory](tx).Where("product_id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrInventoryNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find inventory by product id error: %w", err)
	}

	return &inventory, nil
}

func (i inventoryRepository) GetInventoryLock(ctx context.Context, lockKey string, token string) (bool, error) {
	result, err := i.rds.SetNX(ctx, lockKey, token, 1000*time.Millisecond).Result()

	if err != nil {
		return false, fmt.Errorf("redis: get inventory lock failed: %w", err)
	}

	return result, nil
}

func (i inventoryRepository) ValidateAndUpdateReservedQuantity(
	ctx context.Context,
	keys []string,
	args ...interface{},
) error {
	result, err := redisscript.ValidateAndUpdateReservedQuantityScript.Run(
		ctx,
		i.rds,
		keys,
		args...,
	).Result()

	if err != nil {
		return fmt.Errorf("redis: validate and update reserved quantity failed: %w", err)
	}

	errCode, idx := result.([]interface{})[0].(int64), result.([]interface{})[1].(int64)

	productKey := ""

	// 에러가 발생한 경우만 상품 key 가져옴
	if errCode < 1 {
		productKey = keys[int(idx)-1]
	}

	switch errCode {
	case 0:
		return fmt.Errorf("redis: pKey - %s: %w", productKey, ErrRedisInventoryNotFound)
	case -1:
		return fmt.Errorf("redis: pKey - %s: %w", productKey, ErrRedisInvalidQuantity)
	case -2:
		return fmt.Errorf("redis: pKey - %s: %w", productKey, ErrRedisInsufficientQuantity)

	default:
		return nil
	}
}

func (i inventoryRepository) SetInventory(ctx context.Context, key string, fields map[string]interface{}) error {
	result, err := i.rds.HSet(ctx, key, fields).Result()

	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if result != 1 {
		return fmt.Errorf(" %w", ErrRedisInventoryAlreadyExists)
	}

	return nil
}
