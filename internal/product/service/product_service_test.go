package service

import (
	"context"
	"io"
	"log/slog"
	"order_system/internal/pkg/rediskey"
	productport "order_system/internal/product"
	"order_system/internal/product/domain"
	"testing"

	"gorm.io/gorm"
)

type productServiceFixture struct {
	ctx            context.Context
	svc            *ProductService
	productTx      *fakeProductUnitOfWork
	productRepo    *fakeProductStore
	productCache   *fakeProductCache
	inventoryRepo  *fakeInventoryRepository
	inventoryCache *fakeInventoryCache
}

func newProductServiceFixture() *productServiceFixture {
	productRepo := &fakeProductStore{}
	inventoryRepo := &fakeInventoryRepository{}
	productTx := &fakeProductUnitOfWork{
		productStore:   productRepo,
		inventoryStore: inventoryRepo,
	}
	productCache := &fakeProductCache{}
	inventoryCache := &fakeInventoryCache{}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := NewProductService(
		logger,
		productTx,
		productRepo,
		productCache,
		inventoryCache,
	)

	return &productServiceFixture{
		ctx:            context.Background(),
		svc:            svc,
		productTx:      productTx,
		productRepo:    productRepo,
		productCache:   productCache,
		inventoryRepo:  inventoryRepo,
		inventoryCache: inventoryCache,
	}
}

func TestProductService_CreateProduct(t *testing.T) {
	fixture := newProductServiceFixture()

	description := "테스트 상품 설명"
	req := domain.CreatRequest{
		ProductRequest: domain.ProductRequest{
			Name:        "테스트 상품",
			Description: &description,
			Price:       10000,
			Status:      domain.StatusActive,
		},
		Inventory: domain.InventoryRequest{
			TotalQuantity:    100,
			ReservedQuantity: 0,
			SoldQuantity:     0,
		},
	}

	got, err := fixture.svc.CreateProduct(fixture.ctx, req)

	if err != nil {
		t.Fatalf("CreateProduct error = %v", err)
	}

	if got.ID != 1 {
		t.Fatalf("CreateProduct ID = %d, want 1", got.ID)
	}

	if got.Name != req.Name {
		t.Fatalf("CreateProduct Name = %q, want %q", got.Name, req.Name)
	}

	if got.Description == nil || *got.Description != description {
		t.Fatalf("CreateProduct Description = %v, want %q", got.Description, description)
	}

	if got.Price != req.Price {
		t.Fatalf("CreateProduct Price = %d, want %d", got.Price, req.Price)
	}

	if got.Status != req.Status {
		t.Fatalf("CreateProduct Status = %q, want %q", got.Status, req.Status)
	}

	if got.Inventory.TotalQuantity != req.Inventory.TotalQuantity {
		t.Fatalf("CreateProduct Inventory.TotalQuantity = %d, want %d",
			got.Inventory.TotalQuantity,
			req.Inventory.TotalQuantity,
		)
	}

	if fixture.productRepo.storedProduct == nil {
		t.Fatalf("상품 생성 실패")
	}

	if fixture.inventoryRepo.storedInventory == nil {
		t.Fatalf("재고 생성 실패")
	}

	if fixture.inventoryRepo.storedInventory.ProductID != got.ID {
		t.Fatalf("inventory ProductID = %d, want %d", fixture.inventoryRepo.storedInventory.ProductID, got.ID)
	}

	wantProductCacheKey := rediskey.ProductKey(got.ID)
	if fixture.productCache.storedKey != wantProductCacheKey {
		t.Fatalf("product cache key = %q, want %q", fixture.productCache.storedKey, wantProductCacheKey)
	}

	wantInventoryCacheKey := rediskey.ProductInventoryKey(got.ID)
	if fixture.inventoryCache.storedKey != wantInventoryCacheKey {
		t.Fatalf("inventory cache key = %q, want %q", fixture.inventoryCache.storedKey, wantInventoryCacheKey)
	}
}

func TestProductService_GetProduct(t *testing.T) {

	t.Run("redis cache에 상품과 재고가 있으면 DB를 조회하지 않고 반환한다", func(t *testing.T) {
		fixture := newProductServiceFixture()
		fixture.productCache.findResult = map[string]string{
			"name":        "캐시된 상품",
			"description": "캐시된 상품 설명",
			"price":       "10000",
			"status":      "ACTIVE",
		}
		fixture.inventoryCache.findResult = map[string]string{
			"total_quantity":    "100",
			"reserved_quantity": "0",
			"sold_quantity":     "0",
		}

		got, err := fixture.svc.GetProduct(fixture.ctx, domain.UriRequest{ID: 1})

		if err != nil {
			t.Fatalf("GetProduct() error = %v", err)
		}

		if got.ID != 1 {
			t.Fatalf("GetProduct() ID = %d, want %d", got.ID, 1)
		}

		if got.Name != "캐시된 상품" {
			t.Fatalf("GetProduct() Name = %s, want %s", got.Name, "캐시된 상품")
		}

		if got.Price != 10000 {
			t.Fatalf("GetProduct() Price = %d, want %d", got.Price, 10000)
		}

		if got.Status != domain.StatusActive {
			t.Fatalf("GetProduct() Status = %s, want %s", got.Status, domain.StatusActive)
		}

		if got.Inventory.TotalQuantity != 100 {
			t.Fatalf("GetProduct() TotalQuantity = %d, want %d", got.Inventory.TotalQuantity, 100)
		}

		if got.Inventory.ReservedQuantity != 0 {
			t.Fatalf("GetProduct() ReservedQuantity = %d, want %d", got.Inventory.ReservedQuantity, 0)
		}

		if got.Inventory.SoldQuantity != 0 {
			t.Fatalf("GetProduct() SoldQuantity = %d, want %d", got.Inventory.SoldQuantity, 100)
		}

		if fixture.productRepo.findCalled {
			t.Fatalf("ProductRepository.Find() should not be called")
		}

		if fixture.inventoryRepo.findCalled {
			t.Fatalf("InventoryRepository.FindByProductID() should not be called")
		}
	})

	t.Run("redis에 상품이 없으면 DB 조회 후 cache에 저장한다", func(t *testing.T) {
		fixture := newProductServiceFixture()

		description := "DB 상품 설명"
		fixture.productRepo.findProduct = &domain.Product{
			Model:       gorm.Model{ID: 1},
			Name:        "DB 상품 이름",
			Description: &description,
			Price:       10000,
			Status:      domain.StatusActive,
		}
		fixture.inventoryRepo.findInventory = &domain.Inventory{
			Model:            gorm.Model{ID: 1},
			ProductID:        1,
			TotalQuantity:    100,
			ReservedQuantity: 0,
			SoldQuantity:     0,
		}

		got, err := fixture.svc.GetProduct(fixture.ctx, domain.UriRequest{ID: 1})

		if err != nil {
			t.Fatalf("GetProduct() error = %v", err)
		}

		if fixture.productCache.findResult != nil {
			t.Fatalf("productCache.findResult should be nil")
		}

		if fixture.inventoryCache.findResult != nil {
			t.Fatalf("inventoryCache.findResult should be nil")
		}

		if got.Name != "DB 상품 이름" {
			t.Errorf("GetProduct() Name = %s, want %s", got.Name, "DB 상품 이름")
		}

		if got.Price != 10000 {
			t.Errorf("GetProduct() Price = %d, want %d", got.Price, 10000)
		}

		if got.Inventory.TotalQuantity != 100 {
			t.Errorf("GetProduct() TotalQuantity = %d, want %d", got.Inventory.TotalQuantity, 100)
		}

		if got.Inventory.ReservedQuantity != 0 {
			t.Errorf("GetProduct() ReservedQuantity = %d, want %d", got.Inventory.ReservedQuantity, 0)
		}

		if got.Inventory.SoldQuantity != 0 {
			t.Errorf("GetProduct() SoldQuantity = %d, want %d", got.Inventory.SoldQuantity, 0)
		}

		if !fixture.productRepo.findCalled {
			t.Errorf("GetProduct() productRepo.Find() not called")
		}

		if !fixture.inventoryRepo.findCalled {
			t.Errorf("GetProduct() inventoryRepo.Find() not called")
		}

		wantProductCacheKey := rediskey.ProductKey(1)
		if fixture.productCache.storedKey != wantProductCacheKey {
			t.Errorf("GetProduct() productCacheKey = %s, want %s",
				fixture.productCache.storedKey, wantProductCacheKey)
		}

		wantInventoryCacheKey := rediskey.ProductInventoryKey(1)
		if fixture.inventoryCache.storedKey != wantInventoryCacheKey {
			t.Errorf("GetProduct() inventoryCacheKey = %s, want %s",
				fixture.inventoryCache.storedKey, wantInventoryCacheKey)
		}

	})
}

func TestProductService_UpdateProduct(t *testing.T) {
	fixture := newProductServiceFixture()

	description := "수정 상품 설명"
	dto := domain.UpdateRequest{
		ID: 1,
		ProductRequest: domain.ProductRequest{
			Name:        "수정 상품",
			Description: &description,
			Price:       10000,
			Status:      domain.StatusInactive,
		},
		Inventory: domain.UpdateInventoryRequest{
			TotalQuantity: 90,
		},
	}

	got, err := fixture.svc.UpdateProduct(fixture.ctx, dto)
	if err != nil {
		t.Fatalf("UpdateProduct() error = %v", err)
	}

	if fixture.productRepo.updatedID != dto.ID {
		t.Fatalf("UpdateProduct() ID = %d, want %d", fixture.productRepo.updatedID, dto.ID)
	}

	if fixture.inventoryRepo.updatedID != dto.ID {
		t.Fatalf("UpdateProduct() ID = %d, want %d", fixture.inventoryRepo.updatedID, dto.ID)
	}

	if fixture.productRepo.updatedProduct == nil {
		t.Fatalf("UpdateProduct() updatedProduct is nil")
	}

	if fixture.inventoryRepo.updatedInventory == nil {
		t.Fatalf("UpdateProduct() updatedInventory is nil")
	}

	if fixture.inventoryRepo.updatedInventory.ProductID != dto.ID {
		t.Fatalf("UpdateProduct() ProductID = %d, want %d",
			fixture.inventoryRepo.updatedInventory.ProductID, dto.ID)
	}

	if got.Name != dto.ProductRequest.Name {
		t.Fatalf("UpdateProduct() Name = %s, want %s", got.Name, dto.ProductRequest.Name)
	}

	if got.Description == nil || *got.Description != *dto.ProductRequest.Description {
		t.Fatalf("UpdateProduct() Description = %v, want %v", got.Description, dto.ProductRequest.Description)
	}

	if got.Price != dto.ProductRequest.Price {
		t.Fatalf("UpdateProduct() Price = %d, want %d", got.Price, dto.ProductRequest.Price)
	}

	if got.Status != dto.ProductRequest.Status {
		t.Fatalf("UpdateProduct() Status = %s, want %s", got.Status, dto.ProductRequest.Status)
	}

	if got.Inventory.TotalQuantity != dto.Inventory.TotalQuantity {
		t.Fatalf("UpdateProduct() TotalQuantity = %d, want %d", got.Inventory.TotalQuantity, dto.Inventory.TotalQuantity)
	}

	// 업데이트 성공 후 상품 및 재고 수정사항 레디스에 반영
	wantProductKey := rediskey.ProductKey(dto.ID)
	if fixture.productCache.updatedKey != wantProductKey {
		t.Fatalf("UpdateProduct() product cache key = %s, want %s",
			fixture.productCache.updatedKey, wantProductKey)
	}

	wantInventoryKey := rediskey.ProductInventoryKey(dto.ID)
	if fixture.inventoryCache.updatedKey != wantInventoryKey {
		t.Fatalf("UpdateProduct() inventory cache key = %s, want %s",
			fixture.inventoryCache.updatedKey, wantInventoryKey)
	}

	if fixture.productCache.updatedFields["name"] != dto.ProductRequest.Name {
		t.Fatalf("UpdateProduct() product cache name = %s, want %s",
			fixture.productCache.updatedFields["name"], dto.ProductRequest.Name)
	}

	if fixture.productCache.updatedFields["description"] != *dto.ProductRequest.Description {
		t.Fatalf("UpdateProduct() product cache description = %s, want %v",
			fixture.productCache.updatedFields["description"], *dto.ProductRequest.Description)
	}

	if fixture.productCache.updatedFields["price"] != dto.ProductRequest.Price {
		t.Fatalf("UpdateProduct() product cache price = %d, want %d",
			fixture.productCache.updatedFields["price"], dto.ProductRequest.Price)
	}

	if fixture.productCache.updatedFields["status"] != dto.ProductRequest.Status.String() {
		t.Fatalf("UpdateProduct() product cache status = %s, want %s",
			fixture.productCache.updatedFields["status"], dto.ProductRequest.Status)
	}

	if fixture.inventoryCache.updatedFields["total_quantity"] != dto.Inventory.TotalQuantity {
		t.Fatalf("UpdateProduct() inventory cache total quantity = %v, want %v",
			fixture.inventoryCache.updatedFields["total_quantity"], dto.Inventory.TotalQuantity)
	}

}

func TestProductService_DeleteProduct(t *testing.T) {
	fixture := newProductServiceFixture()
	// db 삭제를 진행
	err := fixture.svc.DeleteProduct(fixture.ctx, domain.UriRequest{ID: 1})

	if err != nil {
		t.Fatalf("DeleteProduct() error = %v", err)
	}

	if fixture.productRepo.deletedID != 1 {
		t.Fatalf("DeleteProduct() product repo deleted id = %d, want %d",
			fixture.productRepo.deletedID, 1)
	}

	// db 삭제에 성공했다면 redis 캐시 삭제
	wantProductCacheKey := rediskey.ProductKey(1)
	if fixture.productCache.deletedKey != wantProductCacheKey {
		t.Fatalf("DeleteProduct() product cache key = %s, want %s",
			fixture.productCache.deletedKey, wantProductCacheKey)
	}

	wantInventoryCacheKey := rediskey.ProductInventoryKey(1)
	if fixture.inventoryCache.deletedKey != wantInventoryCacheKey {
		t.Fatalf("DeleteProduct() inventory cache key = %s, want %s",
			fixture.inventoryCache.deletedKey, wantInventoryCacheKey)
	}

}

type fakeProductStore struct {
	storedProduct  *domain.Product
	updatedProduct *domain.Product
	updatedID      uint
	deletedID      uint
	findProduct    *domain.Product
	findCalled     bool
}

func (f *fakeProductStore) Store(ctx context.Context, product *domain.Product) error {
	product.ID = 1
	f.storedProduct = product
	return nil
}

func (f *fakeProductStore) Find(ctx context.Context, id uint) (*domain.Product, error) {
	f.findCalled = true
	return f.findProduct, nil
}

func (f *fakeProductStore) Update(ctx context.Context, id uint, fields *domain.Product) (*domain.Product, error) {
	f.updatedID = id
	fields.ID = id
	f.updatedProduct = fields
	return f.updatedProduct, nil
}

func (f *fakeProductStore) Delete(ctx context.Context, id uint) error {
	f.deletedID = id
	return nil
}

type fakeInventoryRepository struct {
	storedInventory  *domain.Inventory
	updatedInventory *domain.Inventory
	updatedID        uint
	findInventory    *domain.Inventory
	findCalled       bool
}

func (f *fakeInventoryRepository) Store(ctx context.Context, inventory *domain.Inventory) error {
	f.storedInventory = inventory
	return nil
}

func (f *fakeInventoryRepository) FindByProductID(ctx context.Context, id uint) (*domain.Inventory, error) {
	f.findCalled = true
	return f.findInventory, nil
}

func (f *fakeInventoryRepository) Update(ctx context.Context, id uint, fields *domain.Inventory) (*domain.Inventory, error) {
	f.updatedID = id
	f.updatedInventory = fields
	return f.updatedInventory, nil
}

type fakeProductUnitOfWork struct {
	productStore   productport.ProductStore
	inventoryStore productport.InventoryStore
}

func (f *fakeProductUnitOfWork) Tx(ctx context.Context, txFn func(tx productport.ProductTx) error) error {
	return txFn(&fakeProductTx{
		productStore:   f.productStore,
		inventoryStore: f.inventoryStore,
	})
}

type fakeProductTx struct {
	productStore   productport.ProductStore
	inventoryStore productport.InventoryStore
}

func (f *fakeProductTx) ProductStore() productport.ProductStore {
	return f.productStore
}

func (f *fakeProductTx) InventoryStore() productport.InventoryStore {
	return f.inventoryStore
}

type fakeProductCache struct {
	storedKey     string
	storedFields  map[string]interface{}
	updatedKey    string
	updatedFields map[string]interface{}
	deletedKey    string
	findResult    map[string]string
	findErr       error
}

func (f *fakeProductCache) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	return f.findResult, f.findErr
}

func (f *fakeProductCache) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	f.storedKey = key
	f.storedFields = fields
	return nil
}

func (f *fakeProductCache) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	f.updatedKey = key
	f.updatedFields = fields
	return nil
}

func (f *fakeProductCache) DeleteInRedis(ctx context.Context, key string) error {
	f.deletedKey = key
	return nil
}

type fakeInventoryCache struct {
	storedKey     string
	storedFields  map[string]interface{}
	updatedKey    string
	updatedFields map[string]interface{}
	deletedKey    string
	findResult    map[string]string
	findErr       error
}

func (f *fakeInventoryCache) FindInRedis(ctx context.Context, key string) (map[string]string, error) {
	return f.findResult, f.findErr
}

func (f *fakeInventoryCache) StoreInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	f.storedKey = key
	f.storedFields = fields
	return nil
}

func (f *fakeInventoryCache) UpdateInRedis(ctx context.Context, key string, fields map[string]interface{}) error {
	f.updatedKey = key
	f.updatedFields = fields
	return nil
}

func (f *fakeInventoryCache) DeleteInRedis(ctx context.Context, key string) error {
	f.deletedKey = key
	return nil
}
