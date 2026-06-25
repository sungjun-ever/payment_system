package service

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/auth/domain"
	"order_system/internal/auth/repository"
	"order_system/internal/config"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/pkg/apperr/rediserr"
	"order_system/internal/pkg/hashing"
	"order_system/internal/pkg/token"
	userDomain "order_system/internal/user/domain"
	userRepository "order_system/internal/user/repository"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenAlreadyExist  = errors.New("token already exist")
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService struct {
	authRepo repository.AuthRedisRepository
	userRepo userRepository.UserGormRepository
}

func NewAuthService(
	authRepo repository.AuthRedisRepository,
	userRepo userRepository.UserGormRepository,
) AuthService {
	return AuthService{authRepo, userRepo}
}

func (as *AuthService) ValidUser(ctx context.Context, dto domain.LoginRequest) (*userDomain.User, error) {
	getUser, err := as.userRepo.FindByEmail(ctx, dto.Email)

	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return nil, fmt.Errorf("%w", ErrInvalidCredentials)
		}

		return nil, fmt.Errorf("valid user error: %w", err)
	}

	match := hashing.VerifyPassword(getUser.Password, dto.Password)

	if !match {
		return nil, fmt.Errorf("failed verify password: %w", ErrInvalidCredentials)
	}

	return getUser, nil
}

func (as *AuthService) IssueToken(ctx context.Context, cfg config.Config, user *userDomain.User) (*TokenResponse, error) {
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
		if errors.Is(err, rediserr.ErrConflict) {
			return nil, fmt.Errorf("token: %w, %w", err, ErrTokenAlreadyExist)
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
		return nil, fmt.Errorf("failed parse refresh token: %w", ErrInvalidToken)
	}

	getUser, err := as.userRepo.FindByID(ctx, refreshClaims.UserID)

	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return nil, fmt.Errorf("user email not exist: %w", ErrInvalidCredentials)
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
		if errors.Is(err, rediserr.ErrNotFound) {
			return nil, fmt.Errorf("%w: %w", err, ErrInvalidCredentials)
		}

		if errors.Is(err, repository.ErrTokenMismatch) {
			return nil, fmt.Errorf("%w: %w", err, ErrInvalidToken)
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
