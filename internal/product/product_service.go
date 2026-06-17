package product

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"payment_system/internal/pkg/apperr/dberr"
	"payment_system/internal/pkg/apperr/rediserr"
	"payment_system/internal/pkg/apperr/serviceerr"
	"payment_system/internal/pkg/rediskey"
	productDomain "payment_system/internal/product/domain"
	productRepository "payment_system/internal/product/repository"
	"strconv"

	"gorm.io/gorm"
)

type ProductService struct {
	logger             *slog.Logger
	productGormRepo    productRepository.ProductGormRepository
	productRedisRepo   productRepository.ProductRedisRepository
	inventoryGormRepo  productRepository.InventoryGormRepository
	inventoryRedisRepo productRepository.InventoryRedisRepository
}

func NewProductService(
	logger *slog.Logger,
	productGormRepo productRepository.ProductGormRepository,
	productRedisRepo productRepository.ProductRedisRepository,
	inventoryGormRepo productRepository.InventoryGormRepository,
	inventoryRedisRepo productRepository.InventoryRedisRepository,
) *ProductService {
	return &ProductService{
		logger,
		productGormRepo,
		productRedisRepo,
		inventoryGormRepo,
		inventoryRedisRepo,
	}
}

func (p *ProductService) CreateProduct(
	ctx context.Context,
	dto productDomain.CreatRequest,
) (*productDomain.Resource, error) {
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

	return productDomain.NewResource(products, inventory), nil
}

func (p *ProductService) GetProduct(
	ctx context.Context,
	dto productDomain.UriRequest,
) (*productDomain.Resource, error) {
	var pd *productDomain.Product
	var inven *productDomain.Inventory
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

	return productDomain.NewResource(pd, inven), nil
}

func (p *ProductService) UpdateProduct(
	ctx context.Context,
	dto productDomain.UpdateRequest,
) (*productDomain.Resource, error) {
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

	return productDomain.NewResource(pd, inven), nil
}

func (p *ProductService) DeleteProduct(ctx context.Context, dto productDomain.UriRequest) error {
	err := p.productGormRepo.Delete(ctx, dto.ID)

	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return fmt.Errorf("product id: %d, not found: %w: %w", dto.ID, err, serviceerr.ErrResourceNotFound)
		}
		return fmt.Errorf("delete product failed: %w", err)
	}

	err = p.productRedisRepo.DeleteInRedis(ctx, rediskey.ProductKey(dto.ID))

	if err != nil {
		p.logger.ErrorContext(ctx, "redis delete product failed", "err", err)
	}

	err = p.inventoryRedisRepo.DeleteInRedis(ctx, rediskey.ProductInventoryKey(dto.ID))

	if err != nil {
		p.logger.ErrorContext(ctx, "redis delete inventory failed", "err", err)
	}

	return nil
}

// getProductTransaction 상품, 재고 조회 트랜잭션
func (p *ProductService) getProductTransaction(
	ctx context.Context,
	dto productDomain.UriRequest,
) (pd *productDomain.Product, inven *productDomain.Inventory, err error) {
	err = p.productGormRepo.Transaction(func(tx *gorm.DB) error {
		productRepo := p.productGormRepo.WithTx(tx)
		inventoryRepo := p.inventoryGormRepo.WithTx(tx)

		pd, err = productRepo.Find(ctx, dto.ID)

		if err != nil {
			return err
		}

		inven, err = inventoryRepo.FindByProductID(ctx, dto.ID)

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return nil,
				nil,
				fmt.Errorf("product id: %d, not found: %w: %w", dto.ID, err, serviceerr.ErrResourceNotFound)
		}

		return nil, nil, err
	}

	return pd, inven, nil
}

// getProductInfoFromRedis - 레디스에서 상품 정보 가져와 리턴
func (p *ProductService) getProductInfoFromRedis(
	ctx context.Context,
	dto productDomain.UriRequest,
) (*productDomain.Resource, error) {
	// 레디스에 있는 상품 정보를 가져와 리턴
	productInfos, redisFindProductErr := p.productRedisRepo.FindInRedis(ctx, rediskey.ProductKey(dto.ID))
	inventoryInfos, redisFindInventoryErr := p.inventoryRedisRepo.FindInRedis(ctx, rediskey.ProductInventoryKey(dto.ID))

	var productMapped *productDomain.Resource
	var inventoryMapped *productDomain.InventoryResource

	if redisFindProductErr == nil {
		productMapped, _ = p.redisProductToResource(dto.ID, productInfos)
	} else if !errors.Is(redisFindProductErr, rediserr.ErrEmptyHash) {
		return nil, fmt.Errorf("find product in redis failed: %w", redisFindProductErr)
	}

	if redisFindInventoryErr == nil {
		inventoryMapped, _ = p.redisInventoryToResource(inventoryInfos)
	} else if !errors.Is(redisFindInventoryErr, rediserr.ErrEmptyHash) {
		return nil, fmt.Errorf("find inventory in redis failed: %w", redisFindInventoryErr)
	}

	if productMapped != nil && inventoryMapped != nil {
		productMapped.Inventory = inventoryMapped
		return productMapped, nil
	}

	return nil, nil
}

// createProductTransaction 상품, 재고 생성 DB 트랜잭션
func (p *ProductService) createProductTransaction(
	ctx context.Context,
	products *productDomain.Product,
	inventory *productDomain.Inventory,
) error {
	err := p.productGormRepo.Transaction(func(tx *gorm.DB) error {
		productRepo := p.productGormRepo.WithTx(tx)
		inventoryRepo := p.inventoryGormRepo.WithTx(tx)

		if createProductErr := productRepo.Store(ctx, products); createProductErr != nil {
			return createProductErr
		}

		inventory.ProductID = products.ID

		if createInventoryErr := inventoryRepo.Store(ctx, inventory); createInventoryErr != nil {
			return fmt.Errorf("create inventory failed: %w", createInventoryErr)
		}

		return nil
	})

	return err
}

func (p *ProductService) updateProductTransaction(
	ctx context.Context,
	pid uint,
	product *productDomain.Product,
	inventory *productDomain.Inventory,
) (pd *productDomain.Product, inven *productDomain.Inventory, err error) {
	err = p.productGormRepo.Transaction(func(tx *gorm.DB) error {
		productRepo := p.productGormRepo.WithTx(tx)
		inventoryRepo := p.inventoryGormRepo.WithTx(tx)

		pd, err = productRepo.Update(ctx, pid, product)

		if err != nil {
			if errors.Is(err, dberr.ErrNotFound) {
				return fmt.Errorf("product id: %d, not found: %w: %w", pid, err, serviceerr.ErrResourceNotFound)
			}

			return err
		}

		inventory.ProductID = pd.ID

		inven, err = inventoryRepo.Update(ctx, pid, inventory)

		if err != nil {
			if errors.Is(err, dberr.ErrNotFound) {
				return fmt.Errorf(
					"product id: %d, inventory not found: %w: %w",
					pid,
					err,
					serviceerr.ErrResourceNotFound,
				)
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
func (p *ProductService) storeProductInRedis(
	ctx context.Context,
	products *productDomain.Product,
) {
	err := p.productRedisRepo.StoreInRedis(ctx, rediskey.ProductKey(products.ID), map[string]interface{}{
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
func (p *ProductService) storeInventoryInRedis(
	ctx context.Context,
	productID uint,
	inventory *productDomain.Inventory,
) {
	err := p.inventoryRedisRepo.StoreInRedis(ctx, rediskey.ProductInventoryKey(productID), map[string]interface{}{
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

func (p *ProductService) updateProductInRedis(
	ctx context.Context,
	productID uint,
	product *productDomain.Product,
) {
	err := p.productRedisRepo.UpdateInRedis(ctx, rediskey.ProductKey(productID), map[string]interface{}{
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

func (p *ProductService) updateInventoryInRedis(
	ctx context.Context,
	productID uint,
	inventory *productDomain.Inventory,
) {
	err := p.inventoryRedisRepo.UpdateInRedis(ctx, rediskey.ProductInventoryKey(productID), map[string]interface{}{
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
func (p *ProductService) redisProductToResource(
	id uint,
	infos map[string]string,
) (*productDomain.Resource, error) {
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

	return &productDomain.Resource{
		ID:          id,
		Name:        name,
		Description: redisValueToStringPtr(description),
		Price:       price,
		Status:      productDomain.Status(status),
	}, nil
}

// redisInventoryToResource redis get inventory results to resource mapper
func (p *ProductService) redisInventoryToResource(
	infos map[string]string,
) (*productDomain.InventoryResource, error) {
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

	return &productDomain.InventoryResource{
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
