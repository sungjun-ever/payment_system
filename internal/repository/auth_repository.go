package repository

import (
	"context"
	"errors"
	"payment_system/internal/pkg/rediskey"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID uint) (string, error)
	DeleteRefreshToken(ctx context.Context, userID uint) error
	DeleteSession(ctx context.Context, userID uint) error
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

func (r *authRepository) DeleteSession(ctx context.Context, userID uint) error {
	token, err := r.rds.Get(ctx, rediskey.RefreshToken(userID)).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}

	duration, err := r.rds.TTL(ctx, rediskey.RefreshToken(userID)).Result()

	if err != nil {
		return err
	}

	_, err = r.rds.Del(ctx, rediskey.RefreshToken(userID)).Result()

	if err != nil {
		return err
	}

	if duration == -2 || duration == -1 {
		return nil
	}

	remainingTime := duration * time.Second

	_, err = r.rds.Set(ctx, rediskey.BlackList(token), token, remainingTime).Result()

	if err != nil {
		return err
	}

	return nil
}
