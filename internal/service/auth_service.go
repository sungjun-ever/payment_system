package service

import (
	"context"
	"errors"
	"payment_system/internal/config"
	authDto "payment_system/internal/dto/auth"
	"payment_system/internal/model"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/token"
	"payment_system/internal/repository"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, apperr.NewAppError(apperr.LevelError, 404, apperr.UOO1, "일치하는 사용자가 없음", nil)
		}

		return model.User{}, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "db: find user error", nil)
	}

	return getUser, nil
}

func (as *AuthService) IssueToken(ctx context.Context, cfg config.Config, user model.User) (*TokenResponse, error) {
	accessToken, err := createAccessToken(cfg, user.ID, user.Email)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: create access token error", nil)
	}

	refreshToken, err := createRefreshToken(cfg, user.ID)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: create refresh token error", nil)
	}

	err = as.authRepo.StoreRefreshToken(ctx, refreshToken, user.ID, token.RefreshDuration)

	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: store refresh token error", nil)
		}

		return nil, apperr.NewAppError(apperr.LevelError, 409, apperr.S001, "token: token already exist", nil)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (as *AuthService) RefreshAccessToken(ctx context.Context, cfg config.Config, cookieToken string, claims *token.AccessClaims) (*TokenResponse, error) {
	refreshToken, err := as.authRepo.GetRefreshToken(ctx, claims.UserID)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, apperr.NewAppError(apperr.LevelError, 401, apperr.A003, "token expired", nil)
		}
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: redis error", nil)
	}

	if refreshToken != cookieToken {
		return nil, apperr.NewAppError(apperr.LevelError, 401, apperr.A002, "invalid token", nil)
	}

	err = as.authRepo.DeleteRefreshToken(ctx, claims.UserID)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: redis error", nil)
	}

	accessToken, err := createAccessToken(cfg, claims.UserID, claims.Email)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: create access token error", nil)
	}

	newRefreshToken, err := createRefreshToken(cfg, claims.UserID)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: create refresh token error", nil)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (as *AuthService) DeleteToken(ctx context.Context, accessToken string, claims *token.AccessClaims) error {
	err := as.authRepo.DeleteRefreshToken(ctx, claims.UserID)

	if err != nil {
		return apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: redis error", nil)
	}

	remaining := time.Until(claims.ExpiresAt.Time)

	if remaining > 0 {
		err = as.authRepo.BlacklistAccessToken(ctx, accessToken, remaining)

		if err != nil {
			return apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: redis error", nil)
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
