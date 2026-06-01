package repository

import (
	"context"
	"payment_system/internal/pkg/rediskey"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID uint) (string, error)
	DeleteRefreshToken(ctx context.Context, userID uint) error
	BlacklistAccessToken(ctx context.Context, token string, ttl time.Duration) error
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

func (r *authRepository) GetRefreshToken(ctx context.Context, userID uint) (string, error) {
	token, err := r.rds.Get(ctx, rediskey.RefreshToken(userID)).Result()

	if err != nil {
		return "", err
	}

	return token, nil
}

func (r *authRepository) DeleteRefreshToken(ctx context.Context, userID uint) error {
	_, err := r.rds.Del(ctx, rediskey.RefreshToken(userID)).Result()

	if err != nil {
		return err
	}

	return nil
}

func (r *authRepository) BlacklistAccessToken(ctx context.Context, token string, ttl time.Duration) error {
	err := r.rds.Set(ctx, rediskey.BlackList(token), token, ttl).Err()

	if err != nil {
		return err
	}

	return nil
}
