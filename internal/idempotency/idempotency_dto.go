package idempotency

type UpdateRequest struct {
	Origin       string `json:"origin" binding:"required"`
	Action       string `json:"action" binding:"required"`
	RequestHash  string
	OrderID      *uint
	PaymentID    *uint
	ResponseCode int
	Response     interface{}
}

// CreateRequest 생성 시에 팔요한 것 멱등성 생성 요청 서비스, 관련 액션
type CreateRequest struct {
	Origin string `json:"origin" binding:"required"`
	Action string `json:"action" binding:"required"`
}

type Resource struct {
	Key string `json:"idempotency_key"`
}

func NewResource(key string) *Resource {
	return &Resource{key}
}
