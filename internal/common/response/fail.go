package response

import (
	"github.com/gin-gonic/gin"
)

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type FailResponse struct {
	Success bool       `json:"success,default:false"`
	Error   *ErrorInfo `json:"error"`
}

func ToFailResponse(c *gin.Context, status int, code, message string) {
	response := FailResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}

	c.JSON(status, response)
}
