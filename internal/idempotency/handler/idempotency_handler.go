package handler

import (
	"payment_system/internal/idempotency/domain"
	"payment_system/internal/idempotency/service"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/pkg/token"

	"github.com/gin-gonic/gin"
)

type IdempotencyHandler struct {
	is service.IdempotencyService
}

func NewIdempotencyHandler(is service.IdempotencyService) *IdempotencyHandler {
	return &IdempotencyHandler{is}
}

func (h *IdempotencyHandler) Create(c *gin.Context) {
	var dto domain.CreateRequest

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
