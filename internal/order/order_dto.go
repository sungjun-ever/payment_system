package order

import (
	"fmt"
	"time"
)

type OrderedItem struct {
	ProductID   uint   `json:"product_id" binding:"required,numeric"`
	ProductName string `json:"product_name" binding:"required"`
	UnitPrice   uint64 `json:"unit_price" binding:"required,numeric"`
	Quantity    uint   `json:"quantity" binding:"required,numeric,gt=0"`
	TotalPrice  uint64 `json:"total_price" binding:"required,numeric,gt=0"`
}

type CreateRequest struct {
	UserID       uint          `json:"user_id" binding:"required,numeric"`
	OrderNo      string        `json:"order_no" binding:"required"`
	TotalAmount  uint64        `json:"total_amount" binding:"required,numeric"`
	OrderedAt    string        `json:"ordered_at" binding:"required,datetime=2006-01-02 15:04:05"`
	OrderedItems []OrderedItem `json:"ordered_items" binding:"required,gt=0"`
}

type CreateCommand struct {
	UserID       uint
	OrderNo      string
	TotalAmount  uint64
	OrderedAt    time.Time
	OrderedItems []OrderedItem
}

func (r *CreateRequest) ToCommand() (*CreateCommand, error) {
	loc, err := time.LoadLocation("Asia/Seoul")

	if err != nil {
		return nil, fmt.Errorf("load location error: %w", err)
	}

	orderedAt, err := time.ParseInLocation("2006-01-02 15:04:05", r.OrderedAt, loc)

	if err != nil {
		return nil, fmt.Errorf("parse ordered at error: %w", err)
	}

	return &CreateCommand{
		UserID:       r.UserID,
		OrderNo:      r.OrderNo,
		TotalAmount:  r.TotalAmount,
		OrderedAt:    orderedAt,
		OrderedItems: r.OrderedItems,
	}, nil

}

type Resource struct {
	ID          uint      `json:"id"`
	OrderNo     string    `json:"order_no"`
	Status      Status    `json:"status"`
	TotalAmount uint64    `json:"total_amount"`
	OrderedAt   time.Time `json:"ordered_at,format=2006-01-02 15:04:05"`
}
