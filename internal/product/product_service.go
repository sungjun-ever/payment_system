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

	// 상품 및 재고 생성 db 트랜잭션
	err := p.createProductTransaction(ctx, products, inventory)

	if err != nil {
		return nil, err
	}

	// 생성 정보 레디스에 저장
	p.storeProductInfoInRedis(ctx, products, inventory)

	return NewResource(products, inventory), nil
}

func (p *ProductService) GetProduct(ctx context.Context, dto GetRequest) (*Resource, error) {
	var pd *Product
	var inven *Inventory
	var err error

	// 레디스에 있는 상품 정보를 가져와 리턴
	resource, err := p.getProductInfoFromRedis(ctx, dto)

	if err != nil {
		return nil, err
	}

	if resource != nil {
		return resource, nil
	}

	// 레디스에 없으면 DB에서 가져옴
	pd, inven, err = p.getProductTransaction(ctx, dto)

	// db에서 가져온 정보 레디스에 저장
	p.storeProductInfoInRedis(ctx, pd, inven)

	return NewResource(pd, inven), nil
}

func (p *ProductService) UpdateProduct(ctx context.Context, dto UpdateRequest) ([]*Resource, error) {
	//p.inventoryRepo.
	return nil, nil
}

// getProductTransaction 상품, 재고 조회 트랜잭션
func (p *ProductService) getProductTransaction(
	ctx context.Context,
	dto GetRequest,
) (pd *Product, inven *Inventory, err error) {
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
			return nil, nil, fmt.Errorf("product not found: %w", apperr.ErrResourceNotFound)
		}

		if errors.Is(err, ErrInventoryNotFound) {
			return nil, nil, fmt.Errorf("inventory not found: %w", apperr.ErrResourceNotFound)
		}

		return nil, nil, err
	}

	return pd, inven, nil
}

// getProductInfoFromRedis - 레디스에서 상품 정보 가져와 리턴
func (p *ProductService) getProductInfoFromRedis(ctx context.Context, dto GetRequest) (*Resource, error) {
	// 레디스에 있는 상품 정보를 가져와 리턴
	productInfos, redisFindProductErr := p.productRepo.FindInRedis(ctx, rediskey.ProductKey(dto.ID))
	inventoryInfos, redisFindInventoryErr := p.inventoryRepo.FindInRedis(ctx, rediskey.ProductInventoryKey(dto.ID))

	var productMapped *Resource
	var inventoryMapped *InventoryResource

	if redisFindProductErr == nil {
		productMapped, _ = p.redisProductToResource(dto.ID, productInfos)
	} else if !errors.Is(redisFindProductErr, ErrRedisHashEmpty) {
		return nil, fmt.Errorf("find product in redis failed: %w", redisFindProductErr)
	}

	if redisFindInventoryErr == nil {
		inventoryMapped, _ = p.redisInventoryToResource(inventoryInfos)
	} else if !errors.Is(redisFindInventoryErr, ErrRedisHashEmpty) {
		return nil, fmt.Errorf("find inventory in redis failed: %w", redisFindInventoryErr)
	}

	if productMapped != nil && inventoryMapped != nil {
		productMapped.Inventory = inventoryMapped
		return productMapped, nil
	}

	return nil, nil
}

// createProductTransaction 상품, 재고 생성 DB 트랜잭션
func (p *ProductService) createProductTransaction(ctx context.Context, products *Product, inventory *Inventory) error {
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

	return err
}

// storeProductInfoInRedis 레디스에 상품, 재고 정보 저장
func (p *ProductService) storeProductInfoInRedis(ctx context.Context, products *Product, inventory *Inventory) {
	p.storeProductInRedis(ctx, products)
	p.storeInventoryInRedis(ctx, products.ID, inventory)
}

// storeProductInRedis 레디스에 상품 정보 저장
func (p *ProductService) storeProductInRedis(ctx context.Context, products *Product) {
	_ = p.productRepo.StoreInRedis(ctx, rediskey.ProductKey(products.ID), map[string]interface{}{
		"name":        products.Name,
		"description": stringPtrToRedisValue(products.Description),
		"price":       products.Price,
		"status":      products.Status.String(),
	})
}

// storeInventoryInRedis 레디스에 재고 정보 저장
func (p *ProductService) storeInventoryInRedis(ctx context.Context, productID uint, inventory *Inventory) {
	_ = p.inventoryRepo.StoreInRedis(ctx, rediskey.ProductInventoryKey(productID), map[string]interface{}{
		"total_quantity":    inventory.TotalQuantity,
		"reserved_quantity": inventory.ReservedQuantity,
		"sold_quantity":     inventory.SoldQuantity,
	})
}

// redisProductToResource redis get product results to resource mapper
func (p *ProductService) redisProductToResource(id uint, infos map[string]string) (*Resource, error) {
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

// redisInventoryToResource redis get inventory results to resource mapper
func (p *ProductService) redisInventoryToResource(infos map[string]string) (*InventoryResource, error) {
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
