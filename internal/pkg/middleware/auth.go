package middleware

import (
	"errors"
	"payment_system/internal/config"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/token"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			_ = c.Error(apperr.NewAppError(
				apperr.LevelError,
				401,
				apperr.A002,
				"invalid token",
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
					apperr.LevelError,
					401,
					apperr.A001,
					"token expired",
					nil,
				))
				c.Abort()
				return

			case errors.Is(err, token.ErrInvalidAccessToken):
				_ = c.Error(apperr.NewAppError(apperr.LevelError,
					401,
					apperr.A002,
					"invalid token",
					nil,
				))
				c.Abort()
				return

			default:
				_ = c.Error(apperr.NewAppError(
					apperr.LevelError,
					401,
					apperr.A002,
					"invalid token",
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
