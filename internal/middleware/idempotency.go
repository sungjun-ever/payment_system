package middleware

import (
	"fmt"
	"payment_system/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func IdempotencyKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 헤더에서 키를 가져옴
		key := c.GetHeader("Idempotency-Key")

		// 키가 없는 경우 오류 반환
		if key == "" {
			_ = c.Error(apperr.NewAppError(
				apperr.LevelInfo,
				400,
				apperr.I001,
				fmt.Errorf("idempotency key not found"),
				nil,
			))
			c.Abort()
			return
		}

		c.Set("idempotencyKey", key)
		c.Next()
	}
}
