package middleware

import (
	"errors"
	"payment_system/internal/config"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/rediskey"
	"payment_system/internal/pkg/token"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken = errors.New("invalid access token")
	ErrTokenExpired = errors.New("token expired")
)

func AuthMiddleware(rds *redis.Client, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			_ = c.Error(apperr.NewAppError(
				apperr.LevelError,
				401,
				apperr.A002,
				ErrInvalidToken,
				nil,
			))
			c.Abort()
			return
		}

		result, err := rds.Exists(c.Request.Context(), rediskey.BlackList(tokenString)).Result()

		if err != nil {
			_ = c.Error(apperr.NewAppError(
				apperr.LevelWarn,
				500,
				apperr.S001,
				err,
				nil,
			))
			c.Abort()
			return
		}

		if result == 1 {
			_ = c.Error(apperr.NewAppError(
				apperr.LevelError,
				401,
				apperr.A003,
				ErrTokenExpired,
				nil,
			))
			c.Abort()
			return
		}

		claims, err := token.ParseValidAccessToken(cfg.JwtSecret, tokenString)

		if err != nil {
			switch {
			case errors.Is(err, token.ErrAccessTokenExpired):
				_ = c.Error(apperr.NewAppError(
					apperr.LevelInfo,
					401,
					apperr.A001,
					ErrTokenExpired,
					nil,
				))
				c.Abort()
				return

			case errors.Is(err, token.ErrInvalidAccessToken):
				_ = c.Error(apperr.NewAppError(apperr.LevelInfo,
					401,
					apperr.A002,
					ErrInvalidToken,
					nil,
				))
				c.Abort()
				return

			default:
				_ = c.Error(apperr.NewAppError(
					apperr.LevelError,
					401,
					apperr.A002,
					err,
					nil,
				))
				c.Abort()
				return
			}
		}

		c.Set("accessToken", tokenString)
		c.Set("accessClaims", claims)

		c.Next()
	}
}

func extractBearerToken(header string) (string, error) {
	const prefix = "Bearer "

	if header == "" {
		return "", token.ErrInvalidAccessToken
	}

	if !strings.HasPrefix(header, prefix) {
		return "", token.ErrInvalidAccessToken
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if tokenString == "" {
		return "", token.ErrInvalidAccessToken
	}

	return tokenString, nil
}
