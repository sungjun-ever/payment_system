package product

type InventoryRequest struct {
	TotalQuantity    int `json:"total_quantity" binding:"number"`
	ReservedQuantity int `json:"reserved_quantity" binding:"number"`
	SoldQuantity     int `json:"sold_quantity" binding:"number"`
}

type CreatRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description *string          `json:"description"`
	Price       int64            `json:"price" binding:"required,number"`
	Status      Status           `json:"status"`
	Inventory   InventoryRequest `json:"inventory"`
}

type UpdateRequest struct {
	ID uint `json:"id"`
	CreatRequest
}

type GetRequest struct {
	ID uint `uri:"productID" binding:"required,numeric"`
}

type InventoryResource struct {
	TotalQuantity    int `json:"total_quantity"`
	ReservedQuantity int `json:"reserved_quantity"`
	SoldQuantity     int `json:"sold_quantity"`
}

type Resource struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Price       int64              `json:"price"`
	Status      Status             `json:"status"`
	Inventory   *InventoryResource `json:"inventory"`
}

func NewResource(p *Product, i *Inventory) *Resource {
	return &Resource{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Status:      p.Status,
		Inventory: &InventoryResource{
			TotalQuantity:    i.TotalQuantity,
			ReservedQuantity: i.ReservedQuantity,
			SoldQuantity:     i.SoldQuantity,
		},
	}
}
