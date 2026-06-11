package product

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/rediskey"
	"strconv"

	"gorm.io/gorm"
)

type ProductService struct {
	logger        *slog.Logger
	productRepo   ProductRepository
	inventoryRepo InventoryRepository
}

func NewProductService(logger *slog.Logger, productRepo ProductRepository, inventoryRepo InventoryRepository) *ProductService {
	return &ProductService{logger, productRepo, inventoryRepo}
}

func (p *ProductService) CreateProduct(ctx context.Context, dto CreatRequest) (*Resource, error) {
	products := dto.ToCreateProductEntity()
	inventory := dto.Inventory.ToCreateInventoryEntity()

	// 상품 및 재고 생성 db 트랜잭션
	err := p.createProductTransaction(ctx, products, inventory)

	if err != nil {
		return nil, err
	}

	// 생성 정보 레디스에 저장
	p.storeProductInRedis(ctx, products)
	p.storeInventoryInRedis(ctx, products.ID, inventory)

	return NewResource(products, inventory), nil
}

func (p *ProductService) GetProduct(ctx context.Context, dto UriRequest) (*Resource, error) {
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

	if err != nil {
		return nil, err
	}

	// db에서 가져온 정보 레디스에 저장
	p.storeProductInRedis(ctx, pd)
	p.storeInventoryInRedis(ctx, pd.ID, inven)

	return NewResource(pd, inven), nil
}

func (p *ProductService) UpdateProduct(ctx context.Context, dto UpdateRequest) (*Resource, error) {
	product := dto.ToUpdateProductEntity()
	inventory := dto.Inventory.ToUpdateInventoryEntity()

	pd, inven, err := p.updateProductTransaction(ctx, dto.ID, product, inventory)

	// db 업데이트 실패하면 오류 반환
	if err != nil {
		return nil, err
	}

	// db 업데이트 성공 후 레디스 저장
	p.updateProductInRedis(ctx, pd.ID, pd)
	p.updateInventoryInRedis(ctx, pd.ID, inven)

	return NewResource(pd, inven), nil
}

func (p *ProductService) DeleteProduct(ctx context.Context, dto UriRequest) error {
	err := p.productRepo.Delete(ctx, dto.ID)

	if err != nil {
		return fmt.Errorf("delete product failed: %w", err)
	}

	err = p.productRepo.DeleteInRedis(ctx, rediskey.ProductKey(dto.ID))

	if err != nil {
		p.logger.ErrorContext(ctx, "redis delete product failed", "err", err)
	}

	err = p.inventoryRepo.DeleteInRedis(ctx, rediskey.ProductInventoryKey(dto.ID))

	if err != nil {
		p.logger.ErrorContext(ctx, "redis delete inventory failed", "err", err)
	}

	return nil
}

// getProductTransaction 상품, 재고 조회 트랜잭션
func (p *ProductService) getProductTransaction(
	ctx context.Context,
	dto UriRequest,
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
func (p *ProductService) getProductInfoFromRedis(ctx context.Context, dto UriRequest) (*Resource, error) {
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

		if createInventoryErr := p.inventoryRepo.Store(ctx, tx, inventory); createInventoryErr != nil {
			return fmt.Errorf("create inventory failed: %w", createInventoryErr)
		}

		return nil
	})

	return err
}

func (p *ProductService) updateProductTransaction(
	ctx context.Context,
	pid uint,
	product *Product,
	inventory *Inventory,
) (pd *Product, inven *Inventory, err error) {
	err = p.productRepo.Transaction(func(tx *gorm.DB) error {
		pd, err = p.productRepo.Update(ctx, tx, pid, product)

		if err != nil {
			if errors.Is(err, ErrProductNotFound) {
				return fmt.Errorf("product not found: %w", apperr.ErrResourceNotFound)
			}

			return err
		}

		inventory.ProductID = pd.ID

		inven, err = p.inventoryRepo.Update(ctx, tx, pid, inventory)

		if err != nil {
			if errors.Is(err, ErrInventoryNotFound) {
				return fmt.Errorf("inventory not found: %w", apperr.ErrResourceNotFound)
			}
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("update product transaction failed: %w", err)
	}

	return pd, inven, nil
}

// storeProductInRedis 레디스에 상품 정보 저장
func (p *ProductService) storeProductInRedis(ctx context.Context, products *Product) {
	err := p.productRepo.StoreInRedis(ctx, rediskey.ProductKey(products.ID), map[string]interface{}{
		"name":        products.Name,
		"description": stringPtrToRedisValue(products.Description),
		"price":       products.Price,
		"status":      products.Status.String(),
	})

	if err != nil {
		p.logger.ErrorContext(
			ctx,
			fmt.Sprintf("product id: %d, redis store product failed", products.ID),
			"err",
			err,
		)
	}
}

// storeInventoryInRedis 레디스에 재고 정보 저장
func (p *ProductService) storeInventoryInRedis(ctx context.Context, productID uint, inventory *Inventory) {
	err := p.inventoryRepo.StoreInRedis(ctx, rediskey.ProductInventoryKey(productID), map[string]interface{}{
		"total_quantity":    inventory.TotalQuantity,
		"reserved_quantity": inventory.ReservedQuantity,
		"sold_quantity":     inventory.SoldQuantity,
	})

	if err != nil {
		p.logger.ErrorContext(
			ctx,
			fmt.Sprintf("product id: %d, redis store inventory failed", productID),
			"err",
			err,
		)
	}
}

func (p *ProductService) updateProductInRedis(ctx context.Context, productID uint, product *Product) {
	err := p.productRepo.UpdateInRedis(ctx, rediskey.ProductKey(productID), map[string]interface{}{
		"name":        product.Name,
		"description": stringPtrToRedisValue(product.Description),
		"price":       product.Price,
		"status":      product.Status.String(),
	})

	if err != nil {
		p.logger.ErrorContext(
			ctx,
			fmt.Sprintf("product id: %d, redis update product failed", productID),
			"err",
			err,
		)
	}
}

func (p *ProductService) updateInventoryInRedis(ctx context.Context, productID uint, inventory *Inventory) {
	err := p.inventoryRepo.UpdateInRedis(ctx, rediskey.ProductInventoryKey(productID), map[string]interface{}{
		"total_quantity":    inventory.TotalQuantity,
		"reserved_quantity": inventory.ReservedQuantity,
		"sold_quantity":     inventory.SoldQuantity,
	})

	if err != nil {
		p.logger.ErrorContext(
			ctx,
			fmt.Sprintf("product id: %d, redis update failed", productID),
			"err",
			err,
		)
	}
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
