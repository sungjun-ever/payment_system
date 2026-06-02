package service

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/config"
	authDto "payment_system/internal/dto/auth"
	"payment_system/internal/model"
	"payment_system/internal/pkg/token"
	"payment_system/internal/repository"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenNotFound      = errors.New("token not found")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenConflict      = errors.New("token conflict")
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService struct {
	authRepo repository.AuthRepository
	userRepo repository.UserRepository
}

func NewAuthService(
	authRepo repository.AuthRepository,
	userRepo repository.UserRepository,
) AuthService {
	return AuthService{authRepo, userRepo}
}

func (as *AuthService) ValidUser(ctx context.Context, dto authDto.LoginRequest) (model.User, error) {
	getUser, err := as.userRepo.FindByEmail(ctx, dto.Email)

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, fmt.Errorf("%w", ErrInvalidCredentials)
		}

		return model.User{}, fmt.Errorf("valid user error: %w", err)
	}

	return getUser, nil
}

func (as *AuthService) IssueToken(ctx context.Context, cfg config.Config, user model.User) (*TokenResponse, error) {
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
		if errors.Is(err, repository.ErrTokenAlreadyExists) {
			return nil, fmt.Errorf("%w", ErrTokenConflict)
		}

		return nil, fmt.Errorf("store refresh token error: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (as *AuthService) RefreshAccessToken(ctx context.Context, cfg config.Config, cookieToken string, claims *token.AccessClaims) (*TokenResponse, error) {
	refreshToken, err := as.authRepo.GetRefreshToken(ctx, claims.UserID)

	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, fmt.Errorf("%w", ErrTokenNotFound)
		}
		return nil, fmt.Errorf("get refresh token error: %w", err)
	}

	if refreshToken != cookieToken {
		return nil, fmt.Errorf("%w", ErrInvalidToken)
	}

	err = as.authRepo.DeleteRefreshToken(ctx, claims.UserID)

	if err != nil {
		return nil, fmt.Errorf("delete refresh token error: %w", err)
	}

	accessToken, err := createAccessToken(cfg, claims.UserID, claims.Email)

	if err != nil {
		return nil, fmt.Errorf("create access token error: %w", err)
	}

	newRefreshToken, err := createRefreshToken(cfg, claims.UserID)

	if err != nil {
		return nil, fmt.Errorf("create refresh token error: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (as *AuthService) DeleteToken(ctx context.Context, accessToken string, claims *token.AccessClaims) error {
	err := as.authRepo.DeleteRefreshToken(ctx, claims.UserID)

	if err != nil {
		return fmt.Errorf("delete refresh token error: %w", err)
	}

	remaining := time.Until(claims.ExpiresAt.Time)

	if remaining > 0 {
		err = as.authRepo.BlacklistAccessToken(ctx, accessToken, remaining)

		if err != nil {
			return fmt.Errorf("blacklist access token error: %w", err)
		}
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
