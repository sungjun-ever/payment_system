package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrProductNotFound = errors.New("db: product not found")
	ErrRedisHashEmpty  = errors.New("redis: hash is empty")
)

type ProductRepository interface {
	Transaction(txFn func(tx *gorm.DB) error) error
	Store(ctx context.Context, tx *gorm.DB, product *Product) error
	StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	Update(ctx context.Context, tx *gorm.DB, id uint, fields *Product) (*Product, error)
	UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	Find(ctx context.Context, tx *gorm.DB, id uint) (*Product, error)
	FindInRedis(ctx context.Context, key string) (map[string]string, error)
}

type productRepository struct {
	mysql *gorm.DB
	rds   *redis.Client
}

func NewProductRepository(db *gorm.DB, rds *redis.Client) ProductRepository {
	return &productRepository{db, rds}
}

func (p *productRepository) Transaction(txFn func(tx *gorm.DB) error) error {
	return p.mysql.Transaction(txFn)
}

func (p *productRepository) Store(ctx context.Context, tx *gorm.DB, product *Product) error {
	err := gorm.G[Product](tx).Create(ctx, product)

	if err != nil {
		return fmt.Errorf("db: create product error: %w", err)
	}

	return nil
}

func (p *productRepository) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := p.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: store product in redis error: %w", err)
	}

	return nil
}

func (p *productRepository) Update(ctx context.Context, tx *gorm.DB, id uint, fields *Product) (*Product, error) {
	err := tx.WithContext(ctx).Model(&Product{}).Where("id = ?", id).Updates(fields).Error

	if err != nil {
		return nil, fmt.Errorf("db: update product error: %w", err)
	}

	product, err := p.Find(ctx, tx, id)

	if err != nil {
		return nil, err
	}

	return product, nil
}

func (p *productRepository) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := p.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: update product in redis error: %w", err)
	}

	return nil
}

func (p *productRepository) Find(ctx context.Context, tx *gorm.DB, id uint) (*Product, error) {
	product, err := gorm.G[Product](tx).Where("id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrProductNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find product by id error: %w", err)
	}

	return &product, nil
}

func (p *productRepository) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	results, err := p.rds.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis: find product in redis error: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("%w", ErrRedisHashEmpty)
	}

	return results, nil
}
