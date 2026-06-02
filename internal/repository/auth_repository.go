package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/rediskey"
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

	if errors.Is(err, redis.Nil) {
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
