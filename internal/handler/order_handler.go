package handler

import (
	"fmt"
	"payment_system/internal/dto/order"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/service"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	os service.OrderService
}

func NewOrderHandler(os service.OrderService) *OrderHandler {
	return &OrderHandler{os}
}

func (o *OrderHandler) Create(c *gin.Context) {
	var dto order.CreateRequest

	if err := c.ShouldBindJSON(&dto); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	idempotencyKey, exist := c.Get("idempotencyKey")

	if exist == false {
		_ = c.Error(apperr.NewAppError(
			apperr.LevelWarn,
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
			apperr.LevelWarn,
			400,
			apperr.C001,
			fmt.Errorf("request hash not exists"),
			nil,
		))
		return
	}

	created, err := o.os.CreateOrder(c.Request.Context(), idempotencyKey.(string), requestHash.(string), dto)

	if err != nil {
		_ = c.Error(toAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, created)
}

func (o *OrderHandler) Get(c *gin.Context) {

}
