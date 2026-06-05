package service

import (
	"context"
	"errors"
	"fmt"
	productDto "payment_system/internal/dto/product"
	"payment_system/internal/model"
	"payment_system/internal/repository"

	"gorm.io/gorm"
)

type ProductService struct {
	productRepo   repository.ProductRepository
	inventoryRepo repository.InventoryRepository
}

func NewProductService(productRepo repository.ProductRepository, inventoryRepo repository.InventoryRepository) *ProductService {
	return &ProductService{productRepo, inventoryRepo}
}

func (p *ProductService) CreateProduct(ctx context.Context, dto productDto.CreatRequest) (*productDto.Resource, error) {
	products := &model.Product{
		Name:        dto.Name,
		Description: dto.Description,
		Price:       dto.Price,
		Status:      dto.Status,
	}

	inventory := &model.Inventory{
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

	response := &productDto.Resource{
		ID:          products.ID,
		Name:        products.Name,
		Description: products.Description,
		Price:       products.Price,
		Status:      products.Status,
		Inventory: &productDto.InventoryResource{
			TotalQuantity: inventory.TotalQuantity,
			SoldQuantity:  inventory.SoldQuantity,
		},
	}
	return response, nil
}

func (p *ProductService) GetProduct(ctx context.Context, dto productDto.GetRequest) (*productDto.Resource, error) {
	var pd *model.Product
	var inven *model.Inventory
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
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, fmt.Errorf("product not found: %w", ErrResourceNotFound)
		}

		if errors.Is(err, repository.ErrInventoryNotFound) {
			return nil, fmt.Errorf("inventory not found: %w", ErrResourceNotFound)
		}

		return nil, err
	}

	response := &productDto.Resource{
		ID:          pd.ID,
		Name:        pd.Name,
		Description: pd.Description,
		Price:       pd.Price,
		Status:      pd.Status,
		Inventory: &productDto.InventoryResource{
			TotalQuantity: inven.TotalQuantity,
			SoldQuantity:  inven.SoldQuantity,
		},
	}

	return response, nil
}
