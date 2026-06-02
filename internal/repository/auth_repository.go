package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/rediskey"
	"payment_system/internal/pkg/redisscript"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrTokenAlreadyExists = errors.New("token already exists")
	ErrTokenNotFound      = errors.New("token not found")
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID uint) (string, error)
	DeleteRefreshAndBlacklistAccessToken(ctx context.Context, userID uint, token string, ttl time.Duration) error
	DeleteRefreshToken(ctx context.Context, userID uint) error
}

type authRepository struct {
	rds *redis.Client
}

func NewAuthRepository(rds *redis.Client) AuthRepository {
	return &authRepository{rds}
}

func (r *authRepository) StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error {
	result, err := r.rds.SetNX(ctx, rediskey.RefreshToken(userID), token, ttl).Result()

	if errors.Is(err, redis.Nil) || !result {
		return fmt.Errorf("%w", ErrTokenAlreadyExists)
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *authRepository) GetRefreshToken(ctx context.Context, userID uint) (string, error) {
	token, err := r.rds.Get(ctx, rediskey.RefreshToken(userID)).Result()

	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("%w", ErrTokenNotFound)
	}

	if err != nil {
		return "", err
	}

	return token, nil
}

func (r *authRepository) DeleteRefreshAndBlacklistAccessToken(ctx context.Context, userID uint, token string, ttl time.Duration) error {
	ttlMs := ttl.Milliseconds()

	if ttl > 0 && ttlMs > 0 {
		ttlMs = 1
	}

	err := redisscript.DeleteTokenScript.Run(
		ctx,
		r.rds,
		[]string{rediskey.RefreshToken(userID), rediskey.BlackList(token)},
		ttlMs,
	).Err()

	return err
}

func (r *authRepository) DeleteRefreshToken(ctx context.Context, userID uint) error {
	_, err := r.rds.Del(ctx, rediskey.RefreshToken(userID)).Result()

	if err != nil {
		return err
	}

	return nil
}
