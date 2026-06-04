package product

import "payment_system/internal/model"

type Inventory struct {
	TotalQuantity int `json:"total_quantity" binding:"numeric"`
}

type CreatRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description *string             `json:"description"`
	Price       int64               `json:"price" binding:"required,number"`
	Status      model.ProductStatus `json:"status"`
	Inventory   Inventory           `json:"inventory"`
}

type GetRequest struct {
	ID uint `json:"id" binding:"required,numeric"`
}
