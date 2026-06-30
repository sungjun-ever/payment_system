package gincontext

import (
	"fmt"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/token"

	"github.com/gin-gonic/gin"
)

func GetClaims(c *gin.Context) (*token.AccessClaims, error) {
	claims, exist := c.Get("accessClaims")

	if exist == false {
		return nil, apperr.NewAppError(
			apperr.LevelError,
			400,
			apperr.C001,
			fmt.Errorf("access claims not exists"),
			nil,
		)
	}

	return claims.(*token.AccessClaims), nil
}

func GetIdempotencyKey(c *gin.Context) (string, error) {
	idempotencyKey, exist := c.Get("idempotencyKey")

	if exist == false {
		return "", c.Error(apperr.NewAppError(
			apperr.LevelInfo,
			400,
			apperr.C001,
			fmt.Errorf("idempotency key not exists"),
			nil,
		))
	}

	return idempotencyKey.(string), nil
}

func GetRequestHash(c *gin.Context) (string, error) {
	requestHash, exist := c.Get("request_sha256")

	if exist == false {
		return "", c.Error(apperr.NewAppError(
			apperr.LevelInfo,
			400,
			apperr.C001,
			fmt.Errorf("request hash not exists"),
			nil,
		))
	}

	return requestHash.(string), nil
}
