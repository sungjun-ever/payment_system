package middleware

import (
	"log/slog"
	"payment_system/internal/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestTraceMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		newRequestID, _ := uuid.NewV7()
		requestID := newRequestID.String()

		ctx := logger.WithRequestMeta(c.Request.Context(), requestID)

		c.Request = c.Request.WithContext(ctx)

		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-Id", requestID)

		c.Next()

		latency := time.Since(start)

		log.InfoContext(
			c.Request.Context(),
			"Request completed",
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency/time.Millisecond),
		)
	}
}
