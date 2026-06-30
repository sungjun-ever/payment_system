package domain

import (
	"fmt"
	"time"
)

type OrderedItem struct {
	ProductID   uint   `json:"product_id" binding:"required,numeric"`
	ProductName string `json:"product_name" binding:"required"`
	UnitPrice   uint64 `json:"unit_price" binding:"required,numeric"`
	Quantity    int    `json:"quantity" binding:"required,numeric,gt=0"`
	TotalPrice  uint64 `json:"total_price" binding:"required,numeric,gt=0"`
}

type CreateRequest struct {
	OrderNo      string        `json:"order_no" binding:"required"`
	TotalAmount  uint64        `json:"total_amount" binding:"required,numeric"`
	OrderedAt    string        `json:"ordered_at" binding:"required,datetime=2006-01-02 15:04:05"`
	OrderedItems []OrderedItem `json:"ordered_items" binding:"required,gt=0"`
}

type UriRequest struct {
	ID uint `uri:"orderID" binding:"required,numeric"`
}

type CreateOrderEntity struct {
	UserID      uint
	OrderNo     string
	TotalAmount uint64
	OrderedAt   time.Time
}

type CreateOrderItemsEntity struct {
	ProductID   uint
	ProductName string
	UnitPrice   uint64
	Quantity    uint
	TotalPrice  uint64
}

func (r *CreateRequest) ToCreateOrderEntity(userID uint) (*Order, error) {
	loc, err := time.LoadLocation("Asia/Seoul")

	if err != nil {
		return nil, fmt.Errorf("load location error: %w", err)
	}

	orderedAt, err := time.ParseInLocation("2006-01-02 15:04:05", r.OrderedAt, loc)

	if err != nil {
		return nil, fmt.Errorf("parse ordered at error: %w", err)
	}

	return &Order{
		UserID:      userID,
		OrderNo:     r.OrderNo,
		TotalAmount: r.TotalAmount,
		OrderedAt:   orderedAt,
	}, nil
}

func (r *CreateRequest) ToCreateOrderItemsEntity(orderID uint) []OrderItem {
	var orderItems []OrderItem
	for _, item := range r.OrderedItems {
		orderItems = append(orderItems, OrderItem{
			OrderID:     orderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			TotalPrice:  item.TotalPrice,
		})
	}

	return orderItems
}

type Resource struct {
	ID           uint          `json:"id"`
	OrderNo      string        `json:"order_no"`
	Status       Status        `json:"status"`
	TotalAmount  uint64        `json:"total_amount"`
	OrderedAt    time.Time     `json:"ordered_at,format=2006-01-02 15:04:05"`
	OrderedItems []OrderedItem `json:"ordered_items" binding:"required,gt=0"`
}

type CancelResource struct {
	Message string
}
