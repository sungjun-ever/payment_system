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

func (as *AuthService) IssueToken(ctx context.Context, cfg config.Config, user model.User, refreshDuration time.Duration) (*TokenResponse, error) {
	accessClaims := token.NewAccessClaims(user.ID, user.Email)
	accessToken, err := accessClaims.CreateAccessToken(cfg.JwtSecret)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: create access token error", nil)
	}

	refreshClaims := token.NewRefreshClaims(user.ID, refreshDuration)
	refreshToken, err := refreshClaims.CreateRefreshToken(cfg.JwtSecret)

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 500, apperr.S001, "token: create refresh token error", nil)
	}

	err = as.authRepo.StoreRefreshToken(ctx, refreshToken, user.ID, refreshDuration)

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
