package middleware

import (
	"errors"
	"payment_system/internal/pkg/errUtils"
	"payment_system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func ErrorLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var apiErr *errUtils.ApiError

			if errors.As(err, &apiErr) {
				response.ToFailResponse(c, apiErr.Status, apiErr.Code.String(), apiErr.Message)
			} else {
				response.ToFailResponse(c, 500, errUtils.S001.String(), "Internal Server Error")
			}
		}
	}
}
