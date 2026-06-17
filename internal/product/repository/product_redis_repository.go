package product

import (
	"context"
	"fmt"
	"payment_system/internal/pkg/apperr/rediserr"

	"github.com/redis/go-redis/v9"
)

type ProductRedisRepository struct {
	rds *redis.Client
}

func NewProductRedisRepository(rds *redis.Client) ProductRedisRepository {
	return ProductRedisRepository{rds}
}

func (p *ProductRedisRepository) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	results, err := p.rds.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis: find product in redis error: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("%w", rediserr.ErrEmptyHash)
	}

	return results, nil
}

func (p *ProductRedisRepository) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := p.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: store product in redis error: %w", err)
	}

	return nil
}

func (p *ProductRedisRepository) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := p.rds.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis: update product in redis error: %w", err)
	}

	return nil
}

func (p *ProductRedisRepository) DeleteInRedis(ctx context.Context, key string) error {
	if err := p.rds.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: delete product in redis error: %w", err)
	}

	return nil
}
