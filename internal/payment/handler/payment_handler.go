package handler

import (
	"order_system/internal/payment/domain"
	"order_system/internal/payment/errormap"
	"order_system/internal/payment/service"
	"order_system/internal/pkg/apperr"
	"order_system/internal/pkg/gincontext"
	"order_system/internal/pkg/response"

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

	resource, err := p.ps.CreatePayment(
		c.Request.Context(),
		dto,
		claims,
		idempotencyKey,
		requestHash,
	)

	if err != nil {
		_ = c.Error(errormap.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 201, resource)
}

func (p *PaymentHandler) Refund(c *gin.Context) {
	var uri domain.UriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	var query domain.RefundRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	idempotencyKey, err := gincontext.GetIdempotencyKey(c)

	if err != nil {
		_ = c.Error(err)
		return
	}

	claims, err := gincontext.GetClaims(c)

	if err != nil {
		_ = c.Error(err)
		return
	}

	requestHash, err := gincontext.GetRequestHash(c)

	if err != nil {
		_ = c.Error(err)
		return
	}

	resource, err := p.ps.RefundPayment(
		c.Request.Context(),
		idempotencyKey,
		requestHash,
		uri.PaymentID,
		query.OrderNo,
		claims.UserID,
	)

	if err != nil {
		_ = c.Error(errormap.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 200, resource)
}
