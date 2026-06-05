package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"payment_system/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func HashRequestBodyMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil {
			c.Next()
			return
		}

		body := http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		data, err := io.ReadAll(body)

		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				_ = c.Error(apperr.NewAppError(apperr.LevelError, 413, apperr.C001, err, nil))
				c.Abort()
				return
			}

			_ = c.Error(apperr.NewAppError(apperr.LevelError, 500, apperr.S001, err, nil))
			c.Abort()
			return
		}

		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])

		c.Set("request_sha256", hash)

		c.Request.Body = io.NopCloser(bytes.NewReader(data))

		c.Next()
	}
}
