package model

type PaymentMethod string

const (
	PaymentMethodCard           PaymentMethod = "CARD"
	PaymentMethodVirtualAccount PaymentMethod = "VIRTUAL_ACCOUNT"
	PaymentMethodTransfer       PaymentMethod = "TRANSFER"
	PaymentMethodPoint          PaymentMethod = "POINT"
)
