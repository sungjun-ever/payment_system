package product

import "payment_system/internal/model"

type InventoryResource struct {
	TotalQuantity int `json:"total_quantity"`
	SoldQuantity  int `json:"sold_quantity"`
}

type Resource struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Description *string             `json:"description"`
	Price       int64               `json:"price"`
	Status      model.ProductStatus `json:"status"`
	Inventory   *InventoryResource  `json:"inventory"`
}
