package toss

import (
	"context"
	"math/rand/v2"
	"order_system/internal/pkg/pg"
)

type Status struct {
	code   int
	status string
}

type ResponseDto struct {
	Response       pg.PGResponse
	Reason         string
	PaymentID      string
	IdempotencyKey string
}

type ConfirmationDTO struct {
	PaymentKey string
	OrderId    string
	amount     uint64
}

func ToConfirmationDTO(paymentKey, orderId string, amount uint64) ConfirmationDTO {
	return ConfirmationDTO{paymentKey, orderId, amount}
}

type TossProvider interface {
	Confirm(ctx context.Context, dto ConfirmationDTO) ResponseDto
	Refund() error
	Inquiry(ctx context.Context, orderNo, paymentNo string) ResponseDto
}

type tossProvider struct {
	secretKey string
}

func NewTossProvider(secretKey string) TossProvider {
	return tossProvider{secretKey}
}

// Confirm 실제 요청을 전송하지 않고 랜덤하게 응답을 반환한다.
func (t tossProvider) Confirm(ctx context.Context, dto ConfirmationDTO) ResponseDto {
	s := rand.N(10)

	if s < 5 {
		return ResponseDto{
			pg.Succeeded,
			"",
			"paymentID",
			"idempotencyKey",
		}
	}

	if s == 5 {
		return ResponseDto{
			pg.Completed,
			"completed reason",
			"paymentID",
			"idempotencyKey",
		}
	}

	if s == 6 {
		return ResponseDto{
			pg.Rejected,
			"reject reason",
			"",
			"",
		}
	}

	if s == 7 {
		return ResponseDto{
			pg.ServerFailed,
			"failed reason",
			"",
			"",
		}
	}

	if s == 8 {
		return ResponseDto{
			pg.PGFailed,
			"pg failed reason",
			"",
			"",
		}
	}

	return ResponseDto{
		pg.Unknown,
		"unknown reason",
		"",
		"",
	}
}

func (t tossProvider) Refund() error {
	return nil
}

func (t tossProvider) Inquiry(ctx context.Context, orderNo, paymentNo string) ResponseDto {
	s := rand.N(8)

	if s < 5 {
		return ResponseDto{
			pg.Succeeded,
			"",
			"paymentID",
			"idempotencyKey",
		}
	}

	if s == 5 {
		return ResponseDto{
			pg.Rejected,
			"rejected reason",
			"paymentID",
			"idempotencyKey",
		}
	}

	if s == 6 {
		return ResponseDto{
			pg.PGFailed,
			"pg failed reason",
			"",
			"",
		}
	}

	return ResponseDto{
		pg.Unknown,
		"unknown reason",
		"",
		"",
	}
}
