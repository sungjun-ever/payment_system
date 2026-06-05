package order

import (
	"payment_system/internal/model"
	"time"
)

type Resource struct {
	ID          uint              `json:"id"`
	OrderNo     string            `json:"order_no"`
	Status      model.OrderStatus `json:"status"`
	TotalAmount uint64            `json:"total_amount"`
	OrderedAt   time.Time         `json:"ordered_at,format=2006-01-02 15:04:05"`
}
