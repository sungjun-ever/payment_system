package handler

import (
	"order_system/internal/idempotency/domain"
	"order_system/internal/idempotency/service"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/response"

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

	resource, err := h.is.CreateKey(c.Request.Context(), dto)

	if err != nil {
		_ = c.Error(apperr.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, resource)
}
