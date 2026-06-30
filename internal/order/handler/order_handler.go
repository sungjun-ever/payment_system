package handler

import (
	"order_system/internal/order/domain"
	"order_system/internal/order/errormap"
	"order_system/internal/order/service"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/gincontext"
	"order_system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	os service.OrderService
}

func NewOrderHandler(os service.OrderService) *OrderHandler {
	return &OrderHandler{os}
}

func (o *OrderHandler) Create(c *gin.Context) {
	var dto domain.CreateRequest

	if err := c.ShouldBindJSON(&dto); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	claims, err := gincontext.GetClaims(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	idempotencyKey, err := gincontext.GetIdempotencyKey(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	requestHash, err := gincontext.GetRequestHash(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	created, err := o.os.CreateOrder(
		c.Request.Context(),
		claims,
		idempotencyKey,
		requestHash,
		dto,
	)

	if err != nil {
		_ = c.Error(errormap.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, created)
}

func (o *OrderHandler) Cancel(c *gin.Context) {
	var uri domain.UriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	claims, err := gincontext.GetClaims(c)

	if err != nil {
		_ = c.Error(err)
		return
	}

	resource, err := o.os.CancelOrder(c.Request.Context(), uri, claims.UserID)

	if err != nil {
		_ = c.Error(errormap.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 200, resource)
}
