package middleware

import (
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/idempotency"

	"github.com/gin-gonic/gin"
)

func IdempotencyKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 헤더에서 키를 가져옴
		key := c.GetHeader("Idempotency-Key")

		// 키가 없으면 생성, 원래는 요청 전송 시에 보내준걸 받아야 하지만
		// 테스트를 위해서 없는 경우 미들웨어에서 생성
		if key == "" {
			newKey, err := idempotency.GenerateKey()

			if err != nil {
				_ = c.Error(apperr.NewAppError(apperr.LevelError, 500, apperr.S001, err, nil))
				c.Abort()
				return
			}

			key = newKey
		}

		c.Set("idempotencyKey", key)
		c.Next()
	}
}
