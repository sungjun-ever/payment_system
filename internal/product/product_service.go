package product

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/rediskey"
	"strconv"

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
		TotalQuantity:    dto.Inventory.TotalQuantity,
		ReservedQuantity: dto.Inventory.ReservedQuantity,
		SoldQuantity:     dto.Inventory.SoldQuantity,
	}

	err := p.productRepo.Transaction(func(tx *gorm.DB) error {
		if createProductErr := p.productRepo.Store(ctx, tx, products); createProductErr != nil {
			return createProductErr
		}

		inventory.ProductID = products.ID

		if createInventoryErr := p.inventoryRepo.CreateWithTransaction(ctx, tx, inventory); createInventoryErr != nil {
			return fmt.Errorf("create inventory failed: %w", createInventoryErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// redis에 상품 정보, 재고 별도 저장
	_ = p.productRepo.StoreInRedis(ctx, rediskey.ProductKey(products.ID), map[string]interface{}{
		"name":        products.Name,
		"description": stringPtrToRedisValue(products.Description),
		"price":       products.Price,
		"status":      products.Status.String(),
	})

	_ = p.inventoryRepo.StoreInRedis(ctx, rediskey.ProductInventoryKey(products.ID), map[string]interface{}{
		"total_quantity":    inventory.TotalQuantity,
		"reserved_quantity": inventory.ReservedQuantity,
		"sold_quantity":     inventory.SoldQuantity,
	})

	response := &Resource{
		ID:          products.ID,
		Name:        products.Name,
		Description: products.Description,
		Price:       products.Price,
		Status:      products.Status,
		Inventory: &InventoryResource{
			TotalQuantity:    inventory.TotalQuantity,
			ReservedQuantity: inventory.ReservedQuantity,
			SoldQuantity:     inventory.SoldQuantity,
		},
	}
	return response, nil
}

func (p *ProductService) GetProduct(ctx context.Context, dto GetRequest) (*Resource, error) {
	var pd *Product
	var inven *Inventory
	var err error

	// 레디스에 있는 상품 정보를 가져와 리턴
	productInfos, redisFindProductErr := p.productRepo.FindInRedis(ctx, rediskey.ProductKey(dto.ID))
	inventoryInfos, redisFindInventoryErr := p.inventoryRepo.FindInRedis(ctx, rediskey.ProductInventoryKey(dto.ID))

	var productMapped *Resource
	var inventoryMapped *InventoryResource

	if redisFindProductErr == nil {
		productMapped, _ = p.productInfoMapper(dto.ID, productInfos)
	} else if !errors.Is(redisFindProductErr, ErrRedisHashEmpty) {
		return nil, fmt.Errorf("find product in redis failed: %w", redisFindProductErr)
	}

	if redisFindInventoryErr == nil {
		inventoryMapped, _ = p.inventoryInfoMapper(inventoryInfos)
	} else if !errors.Is(redisFindInventoryErr, ErrRedisHashEmpty) {
		return nil, fmt.Errorf("find inventory in redis failed: %w", redisFindInventoryErr)
	}

	if productMapped != nil && inventoryMapped != nil {
		productMapped.Inventory = inventoryMapped
		return productMapped, nil
	}

	// 레디스에 없으면 DB에서 가져옴
	err = p.productRepo.Transaction(func(tx *gorm.DB) error {
		pd, err = p.productRepo.Find(ctx, tx, dto.ID)

		if err != nil {
			return err
		}

		inven, err = p.inventoryRepo.FindByProductIDWithTransaction(ctx, tx, dto.ID)

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

	// db에서 가져온 정보 레디스에 저장

	_ = p.productRepo.StoreInRedis(ctx, rediskey.ProductKey(dto.ID), map[string]interface{}{
		"name":        pd.Name,
		"description": stringPtrToRedisValue(pd.Description),
		"price":       pd.Price,
		"status":      pd.Status.String(),
	})

	_ = p.inventoryRepo.StoreInRedis(ctx, rediskey.ProductInventoryKey(dto.ID), map[string]interface{}{
		"total_quantity":    inven.TotalQuantity,
		"reserved_quantity": inven.ReservedQuantity,
		"sold_quantity":     inven.SoldQuantity,
	})

	response := &Resource{
		ID:          pd.ID,
		Name:        pd.Name,
		Description: pd.Description,
		Price:       pd.Price,
		Status:      pd.Status,
		Inventory: &InventoryResource{
			TotalQuantity:    inven.TotalQuantity,
			ReservedQuantity: inven.ReservedQuantity,
			SoldQuantity:     inven.SoldQuantity,
		},
	}

	return response, nil
}

func (p *ProductService) UpdateProduct(ctx context.Context, dto UpdateRequest) ([]*Resource, error) {
	//p.inventoryRepo.
	return nil, nil
}

func (p *ProductService) productInfoMapper(id uint, infos map[string]string) (*Resource, error) {
	name := infos["name"]
	description := infos["description"]
	status := infos["status"]
	price, err := strconv.ParseInt(infos["price"], 10, 64)

	if err != nil {
		return nil, fmt.Errorf("parse redis price failed: %w", err)
	}

	if name == "" || status == "" || price == 0 {
		return nil, fmt.Errorf("redis product info is empty")
	}

	return &Resource{
		ID:          id,
		Name:        name,
		Description: redisValueToStringPtr(description),
		Price:       price,
		Status:      Status(status),
	}, nil
}

func (p *ProductService) inventoryInfoMapper(infos map[string]string) (*InventoryResource, error) {
	totalQuantity, err := strconv.ParseInt(infos["total_quantity"], 10, 64)

	if err != nil {
		return nil, fmt.Errorf("parse redis total quantity failed: %w", err)
	}

	reservedQuantity, err := strconv.ParseInt(infos["reserved_quantity"], 10, 64)

	if err != nil {
		return nil, fmt.Errorf("parse redis reserved quantity failed: %w", err)
	}

	soldQuantity, err := strconv.ParseInt(infos["sold_quantity"], 10, 64)

	if err != nil {
		return nil, fmt.Errorf("parse redis sold quantity failed: %w", err)
	}

	return &InventoryResource{
		TotalQuantity:    int(totalQuantity),
		ReservedQuantity: int(reservedQuantity),
		SoldQuantity:     int(soldQuantity),
	}, nil
}

func stringPtrToRedisValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func redisValueToStringPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
