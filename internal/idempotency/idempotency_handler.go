package idempotency

import (
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/pkg/token"

	"github.com/gin-gonic/gin"
)

type IdempotencyHandler struct {
	is IdempotencyService
}

func NewIdempotencyHandler(is IdempotencyService) *IdempotencyHandler {
	return &IdempotencyHandler{is}
}

func (h *IdempotencyHandler) Create(c *gin.Context) {
	var dto CreateRequest

	if err := c.ShouldBindJSON(&dto); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	claims, _ := c.Get("accessClaims")

	resource, err := h.is.CreateKey(c.Request.Context(), dto, claims.(*token.AccessClaims))

	if err != nil {
		_ = c.Error(apperr.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, resource)
}
