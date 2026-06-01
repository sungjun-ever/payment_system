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
		accessToken := c.GetHeader("Authorization")

		if accessToken == "" {
			_ = c.Error(apperr.NewAppError(apperr.LevelError, 401, apperr.A002, "invalid token", nil))
			c.Abort()
			return
		}

		tokenString := strings.Split(accessToken, "Bearer ")[1]
		claims, err := token.ParseValidAccessToken(cfg.JwtSecret, tokenString)

		if err != nil {
			switch {
			case errors.Is(err, token.ErrAccessTokenExpired):
				_ = c.Error(apperr.NewAppError(apperr.LevelError, 401, apperr.A001, "token expired", nil))
				c.Abort()
				return

			case errors.Is(err, token.ErrInvalidAccessToken):
				_ = c.Error(apperr.NewAppError(apperr.LevelError, 401, apperr.A002, "invalid token", nil))
				c.Abort()
				return

			default:
				_ = c.Error(apperr.NewAppError(apperr.LevelError, 401, apperr.A002, "invalid token", nil))
				c.Abort()
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
