package response

import (
	"payment_system/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type ErrorInfo struct {
	Code    apperr.ErrorCode `json:"code"`
	Message string           `json:"message"`
}

type FailResponse struct {
	Success bool       `json:"success,default:false"`
	Error   *ErrorInfo `json:"error"`
}

func ToFailResponse(c *gin.Context, status int, code apperr.ErrorCode, message string) {
	response := FailResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}

	c.JSON(status, response)
}
