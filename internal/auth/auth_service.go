package auth

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr"

	"payment_system/internal/config"
	"payment_system/internal/pkg/hashing"
	"payment_system/internal/pkg/token"
	"payment_system/internal/user"
	"time"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService struct {
	authRepo AuthRepository
	userRepo user.UserRepository
}

func NewAuthService(
	authRepo AuthRepository,
	userRepo user.UserRepository,
) AuthService {
	return AuthService{authRepo, userRepo}
}

func (as *AuthService) ValidUser(ctx context.Context, dto LoginRequest) (*user.User, error) {
	getUser, err := as.userRepo.FindByEmail(ctx, dto.Email)

	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, fmt.Errorf("%w", apperr.ErrInvalidCredentials)
		}

		return nil, fmt.Errorf("valid user error: %w", err)
	}

	match := hashing.VerifyPassword(getUser.Password, dto.Password)

	if !match {
		return nil, fmt.Errorf("failed verify password: %w", apperr.ErrInvalidCredentials)
	}

	return getUser, nil
}

func (as *AuthService) IssueToken(ctx context.Context, cfg config.Config, user *user.User) (*TokenResponse, error) {
	accessToken, err := createAccessToken(cfg, user.ID, user.Email)

	if err != nil {
		return nil, fmt.Errorf("create access token error: %w", err)
	}

	refreshToken, err := createRefreshToken(cfg, user.ID)

	if err != nil {
		return nil, fmt.Errorf("create refresh token error: %w", err)
	}

	err = as.authRepo.StoreRefreshToken(ctx, refreshToken, user.ID, token.RefreshDuration)

	if err != nil {
		if errors.Is(err, ErrTokenAlreadyExists) {
			return nil, fmt.Errorf("token: %w", apperr.ErrConflict)
		}

		return nil, fmt.Errorf("store refresh token error: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (as *AuthService) RotateToken(ctx context.Context, cfg config.Config, cookieToken string) (*TokenResponse, error) {
	refreshClaims, err := token.ParseValidRefreshToken(cfg.JwtSecret, cookieToken)

	if err != nil {
		return nil, fmt.Errorf("failed parse refresh token: %w", apperr.ErrInvalidToken)
	}

	getUser, err := as.userRepo.FindByID(ctx, refreshClaims.UserID)

	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, fmt.Errorf("user email not exist: %w", apperr.ErrInvalidToken)
		}

		return nil, fmt.Errorf("find user error: %w", err)
	}

	accessToken, err := createAccessToken(cfg, getUser.ID, getUser.Email)

	if err != nil {
		return nil, fmt.Errorf("create access token error: %w", err)
	}

	newRefreshToken, err := createRefreshToken(cfg, getUser.ID)

	if err != nil {
		return nil, fmt.Errorf("create refresh token error: %w", err)
	}

	err = as.authRepo.RotateRefreshToken(ctx, getUser.ID, cookieToken, newRefreshToken, token.RefreshDuration)

	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, fmt.Errorf("%w", apperr.ErrTokenNotFound)
		}

		if errors.Is(err, ErrInvalidToken) {
			return nil, fmt.Errorf("%w", apperr.ErrInvalidToken)
		}

		return nil, fmt.Errorf("rotate refresh token error: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (as *AuthService) DeleteToken(ctx context.Context, accessToken string, claims *token.AccessClaims) error {
	remaining := time.Until(claims.ExpiresAt.Time)

	err := as.authRepo.DeleteRefreshAndBlacklistAccessToken(ctx, claims.UserID, accessToken, remaining)

	if err != nil {
		return fmt.Errorf("delete token error: %w", err)
	}

	return nil
}

func createAccessToken(cfg config.Config, userID uint, email string) (string, error) {
	accessClaims := token.NewAccessClaims(userID, email)
	return accessClaims.CreateAccessToken(cfg.JwtSecret)
}

func createRefreshToken(cfg config.Config, userID uint) (string, error) {
	refreshClaims := token.NewRefreshClaims(userID)
	return refreshClaims.CreateRefreshToken(cfg.JwtSecret)
}
