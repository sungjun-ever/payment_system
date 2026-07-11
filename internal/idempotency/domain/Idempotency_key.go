package domain

import "gorm.io/gorm"

type Scope string
type Status string

const (
	ScopeOrderCreated   Scope = "ORDER_CREATED"
	ScopeOrderCancelled Scope = "ORDER_CANCELLED"
	ScopePayOrder       Scope = "PAY_ORDER"
	ScopeRefundOrder    Scope = "REFUND_ORDER"

	StatusSucceeded  Status = "SUCCEEDED"
	StatusFailed     Status = "FAILED"
	StatusProcessing Status = "PROCESSING"
	StatusCancelled  Status = "CANCELLED"
)

type IdempotencyKey struct {
	gorm.Model
	// userid, scope, key를 묶어서 하나의 유니크 인덱스로 설정
	UserID uint   `gorm:"not null;uniqueIndex:usk;column:user_id"`
	Scope  Scope  `gorm:"type:varchar(255);not null;uniqueIndex:usk;column:scope"`
	Key    string `gorm:"type:varchar(255);not null;uniqueIndex:usk;column:key"`

	// 주문 생성시 발생한 요청을 해쉬화하여 저장, 같은 요청이 들어오는 경우 확인
	RequestHash string `gorm:"type:char(64);not null;column:request_hash"`

	Status Status `gorm:"type:varchar(50);index;not null;column:status"`

	OrderID   *uint `gorm:"column:order_id"`
	PaymentID *uint `gorm:"column:payment_id"`

	// 이미 저장된 주문의 경우 동일한 응답을 리턴하기 위해
	// response 정보 저장
	ResponseCode int     `gorm:"column:response_code"`
	ResponseBody *string `gorm:"type:json;column:response_body"`
}
