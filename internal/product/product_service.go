package product

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr"

	"gorm.io/gorm"
)

type ProductService struct {
	productRepo   ProductRepository
	inventoryRepo InventoryRepository
}

func NewProductService(productRepo ProductRepository, inventoryRepo InventoryRepository) *ProductService {
	return &ProductService{productRepo, inventoryRepo}
}

func (p *ProductService) CreateProduct(ctx context.Context, dto CreatRequest) (*Resource, error) {
	products := &Product{
		Name:        dto.Name,
		Description: dto.Description,
		Price:       dto.Price,
		Status:      dto.Status,
	}

	inventory := &Inventory{
		TotalQuantity: dto.Inventory.TotalQuantity,
	}

	err := p.productRepo.Transaction(func(tx *gorm.DB) error {
		if createProductErr := p.productRepo.Store(ctx, tx, products); createProductErr != nil {
			return createProductErr
		}

		inventory.ProductID = products.ID

		if createInventoryErr := p.inventoryRepo.Create(ctx, tx, inventory); createInventoryErr != nil {
			return createInventoryErr
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	response := &Resource{
		ID:          products.ID,
		Name:        products.Name,
		Description: products.Description,
		Price:       products.Price,
		Status:      products.Status,
		Inventory: &InventoryResource{
			TotalQuantity: inventory.TotalQuantity,
			SoldQuantity:  inventory.SoldQuantity,
		},
	}
	return response, nil
}

func (p *ProductService) GetProduct(ctx context.Context, dto GetRequest) (*Resource, error) {
	var pd *Product
	var inven *Inventory
	var err error
	err = p.productRepo.Transaction(func(tx *gorm.DB) error {
		pd, err = p.productRepo.Find(ctx, tx, dto.ID)

		if err != nil {
			return err
		}

		inven, err = p.inventoryRepo.FindByProductID(ctx, tx, dto.ID)

		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			return nil, fmt.Errorf("product not found: %w", apperr.ErrResourceNotFound)
		}

		if errors.Is(err, ErrInventoryNotFound) {
			return nil, fmt.Errorf("inventory not found: %w", apperr.ErrResourceNotFound)
		}

		return nil, err
	}

	response := &Resource{
		ID:          pd.ID,
		Name:        pd.Name,
		Description: pd.Description,
		Price:       pd.Price,
		Status:      pd.Status,
		Inventory: &InventoryResource{
			TotalQuantity: inven.TotalQuantity,
			SoldQuantity:  inven.SoldQuantity,
		},
	}

	return response, nil
}
