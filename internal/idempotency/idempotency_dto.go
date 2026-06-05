package idempotency

type CreateRequest struct {
	UserID       uint
	Scope        Scope
	Key          string
	Status       Status
	RequestHash  string
	OrderID      *uint
	PaymentID    *uint
	ResponseCode int
	Response     interface{}
}
