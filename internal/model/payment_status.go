package model

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusRequested PaymentStatus = "REQUESTED"
	PaymentStatusSucceeded PaymentStatus = "SUCCEEDED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusCanceled  PaymentStatus = "CANCELED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
)
