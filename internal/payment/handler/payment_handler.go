package handler

import (
	"fmt"
	"order_system/internal/payment/domain"
	"order_system/internal/payment/errormap"
	"order_system/internal/payment/service"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/response"
	"order_system/internal/pkg/token"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	ps service.PaymentService
}

func NewPaymentHandler(ps service.PaymentService) *PaymentHandler {
	return &PaymentHandler{ps}
}

func (p *PaymentHandler) Create(c *gin.Context) {
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

	resource, err := p.ps.CreatePayment(
		c.Request.Context(),
		dto,
		claims.(*token.AccessClaims),
		idempotencyKey.(string),
		requestHash.(string),
	)

	if err != nil {
		_ = c.Error(errormap.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, resource)
}
