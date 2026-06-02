package middleware

import (
	"errors"
	"log/slog"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func ErrorLogMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var apiErr *apperr.AppError

			requestID, _ := c.Get("request_id")

			log.InfoContext(
				c.Request.Context(),
				"Request Failed",
				slog.String("error", err.Error()),
				slog.String("request_id", requestID.(string)),
			)

			if errors.As(err, &apiErr) {
				response.ToFailResponse(c, apiErr.Status, apiErr.Code, apiErr.Code.String())
			} else {
				response.ToFailResponse(c, 500, apperr.S001, "Internal Server Error")
			}
		}
	}
}
