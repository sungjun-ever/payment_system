package repository

import (
	"context"
	"payment_system/internal/pkg/rediskey"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error
}

type authRepository struct {
	rds *redis.Client
}

func NewAuthRepository(rds *redis.Client) AuthRepository {
	return &authRepository{rds}
}

func (r *authRepository) StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error {
	_, err := r.rds.SetNX(ctx, rediskey.RefreshToken(userID), token, ttl).Result()

	if err != nil {
		return err
	}

	return nil
}
