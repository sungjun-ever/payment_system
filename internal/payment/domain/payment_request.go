package domain

import (
	"order_system/internal/pkg/pg"
)

type CreateRequest struct {
	PaymentNo string        `json:"payment_no"`
	OrderID   uint          `json:"order_id"`
	Method    Method        `json:"method"`
	Action    AttemptAction `json:"action"`
	Amount    uint64        `json:"amount" binding:"required,number,gte=0"`
	Provider  pg.Channel    `json:"provider"`
}

func (r *CreateRequest) ToCreatePaymentEntity(userID uint) *Payment {
	return &Payment{
		UserID:    userID,
		PaymentNo: r.PaymentNo,
		OrderID:   r.OrderID,
		Status:    PaidProcessing,
	}
}

func (r *CreateRequest) ToCreateAttemptEntity(
	paymentID uint,
	clientIdempotencyKey string,
) *PaymentAttempt {
	return &PaymentAttempt{
		PaymentID:            paymentID,
		ClientIdempotencyKey: clientIdempotencyKey,
		Action:               r.Action,
		Method:               r.Method,
		Status:               AttemptStatusPending,
		Amount:               r.Amount,
		Provider:             r.Provider,
	}
}

type UriRequest struct {
	PaymentID uint `uri:"paymentID" binding:"required,number,gte=0"`
}

type RefundRequest struct {
	OrderNo string `form:"order_no" binding:"required"`
}
