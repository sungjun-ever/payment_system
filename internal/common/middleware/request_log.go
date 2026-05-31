package middleware

import (
	"log/slog"
	"payment_system/internal/common/logger"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLoggerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetHeader("X-Request-Id")
		ctx := logger.WithRequestMeta(c.Request.Context(), requestID)

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)

		log.InfoContext(
			c.Request.Context(),
			"Request completed",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency/time.Millisecond),
		)
	}
}
