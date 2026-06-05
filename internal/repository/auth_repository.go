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
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token string, userID uint, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID uint) (string, error)
	DeleteRefreshAndBlacklistAccessToken(ctx context.Context, userID uint, token string, ttl time.Duration) error
	RotateRefreshToken(ctx context.Context, userID uint, cookieToken string, newRefreshToken string, ttl time.Duration) error
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
		return fmt.Errorf("redis: store refresh token failed: %w", ErrTokenAlreadyExists)
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *authRepository) GetRefreshToken(ctx context.Context, userID uint) (string, error) {
	refreshToken, err := r.rds.Get(ctx, rediskey.RefreshToken(userID)).Result()

	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("redis: get refresh token failed: %w", ErrTokenNotFound)
	}

	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

func (r *authRepository) DeleteRefreshAndBlacklistAccessToken(ctx context.Context, userID uint, token string, ttl time.Duration) error {
	ttlMs := ttl.Milliseconds()

	if ttl > 0 && ttlMs == 0 {
		ttlMs = 1
	}

	err := redisscript.DeleteTokenAndBlacklistScript.Run(
		ctx,
		r.rds,
		[]string{rediskey.RefreshToken(userID), rediskey.BlackList(token)},
		ttlMs,
	).Err()

	return fmt.Errorf("redis: delete refresh and blacklist access token failed: %w", err)
}

func (r *authRepository) RotateRefreshToken(
	ctx context.Context,
	userID uint,
	cookieToken string,
	newRefreshToken string,
	ttl time.Duration,
) error {
	result, err := redisscript.RotateRefreshTokenScript.Run(
		ctx,
		r.rds,
		[]string{rediskey.RefreshToken(userID)},
		cookieToken,
		newRefreshToken,
		ttl.Milliseconds(),
	).Int()

	if err != nil {
		return err
	}

	switch result {
	case 0:
		return fmt.Errorf("redis: refresh token not found: %w", ErrTokenNotFound)
	case -1:
		return fmt.Errorf("redis: refresh and cookie token not same: %w", ErrInvalidToken)
	case 1:
		return nil
	default:
		return fmt.Errorf("redis: unexpected rotate refresh token result: %d", result)
	}
}
