package product

type ProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Price       int64   `json:"price" binding:"required,number,gte=0"`
	Status      Status  `json:"status"`
}

type InventoryRequest struct {
	TotalQuantity    int `json:"total_quantity" binding:"number,gte=0"`
	ReservedQuantity int `json:"reserved_quantity" binding:"number,gte=0"`
	SoldQuantity     int `json:"sold_quantity" binding:"number,gte=0"`
}

type CreatRequest struct {
	ProductRequest
	Inventory InventoryRequest `json:"inventory"`
}

type UpdateInventoryRequest struct {
	TotalQuantity int `json:"total_quantity" binding:"number,gte=0"`
}

type UpdateRequest struct {
	ID uint `json:"-"`
	ProductRequest
	Inventory UpdateInventoryRequest `json:"inventory"`
}

type UriRequest struct {
	ID uint `uri:"productID" binding:"required,numeric"`
}

type InventoryResource struct {
	TotalQuantity    int `json:"total_quantity"`
	ReservedQuantity int `json:"reserved_quantity"`
	SoldQuantity     int `json:"sold_quantity"`
}

func (r *CreatRequest) ToCreateProductEntity() *Product {
	return &Product{
		Name:        r.Name,
		Description: r.Description,
		Price:       r.Price,
		Status:      r.Status,
	}
}

func (r *InventoryRequest) ToCreateInventoryEntity() *Inventory {
	return &Inventory{
		TotalQuantity:    r.TotalQuantity,
		ReservedQuantity: r.ReservedQuantity,
		SoldQuantity:     r.SoldQuantity,
	}
}

func (r *UpdateRequest) ToUpdateProductEntity() *Product {
	return &Product{
		Name:        r.Name,
		Description: r.Description,
		Price:       r.Price,
		Status:      r.Status,
	}
}

func (r *UpdateInventoryRequest) ToUpdateInventoryEntity() *Inventory {
	return &Inventory{
		TotalQuantity: r.TotalQuantity,
	}
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
