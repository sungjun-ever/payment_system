package middleware

import (
	"errors"
	"payment_system/internal/common/errUtils"
	"payment_system/internal/common/response"

	"github.com/gin-gonic/gin"
)

func ErrorLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var apiErr *errUtils.ApiError

			if errors.As(err, &apiErr) {
				response.ToResponse(c, apiErr.Status, apiErr.Code.String(), apiErr.Message)
			} else {
				response.ToResponse(c, 500, errUtils.S001.String(), "Internal Server Error")
			}
		}
	}
}
