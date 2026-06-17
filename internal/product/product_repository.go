package product

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr/dberr"
	"payment_system/internal/pkg/apperr/rediserr"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ProductRepository interface {
	WithTx(tx *gorm.DB) ProductRepository
	Transaction(txFn func(tx *gorm.DB) error) error
	Store(ctx context.Context, product *Product) error
	StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	Update(ctx context.Context, id uint, fields *Product) (*Product, error)
	UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error
	Find(ctx context.Context, id uint) (*Product, error)
	FindInRedis(ctx context.Context, key string) (map[string]string, error)
	Delete(ctx context.Context, id uint) error
	DeleteInRedis(ctx context.Context, key string) error
}

type productRepository struct {
	mysql *gorm.DB
	rds   *redis.Client
}

func NewProductRepository(db *gorm.DB, rds *redis.Client) ProductRepository {
	return &productRepository{db, rds}
}

func (p *productRepository) WithTx(tx *gorm.DB) ProductRepository {
	return &productRepository{tx, p.rds}
}

func (p *productRepository) Transaction(txFn func(tx *gorm.DB) error) error {
	return p.mysql.Transaction(txFn)
}

func (p *productRepository) Store(ctx context.Context, product *Product) error {
	result := p.mysql.WithContext(ctx).Model(&Product{}).Create(product)

	if result.Error != nil {
		return fmt.Errorf("db: create product error: %w", result.Error)
	}

	return nil
}

func (p *productRepository) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := p.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: store product in redis error: %w", err)
	}

	return nil
}

func (p *productRepository) Update(ctx context.Context, id uint, fields *Product) (*Product, error) {
	result := p.mysql.WithContext(ctx).Model(&Product{}).Where("id = ?", id).Updates(fields)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product not found: %w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: update product error: %w", result.Error)
	}

	product, err := p.Find(ctx, id)

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

func (p *productRepository) Find(ctx context.Context, id uint) (*Product, error) {
	var product Product

	result := p.mysql.WithContext(ctx).Model(&Product{}).Where("id = ?", id).First(ctx)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product find error: %w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: find product by id error: %w", result.Error)
	}

	return &product, nil
}

func (p *productRepository) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	results, err := p.rds.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis: find product in redis error: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("%w", rediserr.ErrEmptyHash)
	}

	return results, nil
}

func (p *productRepository) Delete(ctx context.Context, id uint) error {
	result := p.mysql.WithContext(ctx).Model(&Product{}).Where("id = ?", id).Delete(ctx)

	if result.Error != nil {
		return fmt.Errorf("db: delete product error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("deleted product not found: %w", dberr.ErrNotFound)
	}
	
	return nil
}

func (p *productRepository) DeleteInRedis(ctx context.Context, key string) error {
	if err := p.rds.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: delete product in redis error: %w", err)
	}

	return nil
}
