package handler

import (
	"fmt"
	"order_system/internal/order/domain"
	"order_system/internal/order/errormap"
	"order_system/internal/order/service"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/response"
	"order_system/internal/pkg/token"

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

	claims, exist := c.Get("accessClaims")

	if exist == false {
		_ = c.Error(apperr.NewAppError(
			apperr.LevelInfo,
			400,
			apperr.C001,
			fmt.Errorf("access claims not exists"),
			nil,
		))
		return
	}

	idempotencyKey, exist := c.Get("idempotencyKey")

	if exist == false {
		_ = c.Error(apperr.NewAppError(
			apperr.LevelInfo,
			400,
			apperr.C001,
			fmt.Errorf("idempotency key not exists"),
			nil,
		))
		return
	}

	requestHash, exist := c.Get("request_sha256")

	if exist == false {
		_ = c.Error(apperr.NewAppError(
			apperr.LevelInfo,
			400,
			apperr.C001,
			fmt.Errorf("request hash not exists"),
			nil,
		))
		return
	}

	created, err := o.os.CreateOrder(
		c.Request.Context(),
		claims.(*token.AccessClaims),
		idempotencyKey.(string),
		requestHash.(string),
		dto,
	)

	if err != nil {
		_ = c.Error(errormap.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, created)
}

func (o *OrderHandler) Get(c *gin.Context) {

}
