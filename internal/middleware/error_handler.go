package middleware

import (
	"context"
	"errors"
	"log/slog"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func ErrorHandlerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var appError *apperr.AppError

			requestID, _ := c.Get("request_id")

			errorLogByLevel(log, c.Request.Context(), err, appError, requestID.(string))

			if errors.As(err, &appError) {
				response.ToFailResponse(c, appError.Status, appError.Code, appError.Code.String())
			} else {
				response.ToFailResponse(c, 500, apperr.S001, "Internal Server Error")
			}
		}
	}
}

func errorLogByLevel(log *slog.Logger, ctx context.Context, err error, appError *apperr.AppError, requestID string) {
	if errors.As(err, &appError) {
		switch appError.Level {
		case apperr.LevelError:
			log.ErrorContext(ctx, "Request Failed", slog.String("error", err.Error()), slog.String("request_id", requestID))
		case apperr.LevelWarn:
			log.WarnContext(ctx, "Request Failed", slog.String("error", err.Error()), slog.String("request_id", requestID))
		default:
			log.InfoContext(ctx, "Request Failed", slog.String("error", err.Error()), slog.String("request_id", requestID))
		}
	} else {
		log.ErrorContext(ctx, "Request Failed", slog.String("error", err.Error()), slog.String("request_id", requestID))
	}
}
