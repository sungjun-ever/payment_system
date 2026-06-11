package product

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrProductNotFound = errors.New("db: product not found")
)

type ProductRepository interface {
	Transaction(txFn func(tx *gorm.DB) error) error
	Store(ctx context.Context, tx *gorm.DB, product *Product) error
	Find(ctx context.Context, tx *gorm.DB, id uint) (*Product, error)
}

type productRepository struct {
	mysql *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db}
}

func (p *productRepository) Transaction(txFn func(tx *gorm.DB) error) error {
	return p.mysql.Transaction(txFn)
}

func (p *productRepository) Store(ctx context.Context, tx *gorm.DB, product *Product) error {
	err := gorm.G[Product](tx).Create(ctx, product)

	if err != nil {
		return fmt.Errorf("db: create product error: %w", err)
	}

	return nil
}

func (p *productRepository) Find(ctx context.Context, tx *gorm.DB, id uint) (*Product, error) {
	product, err := gorm.G[Product](tx).Where("id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrProductNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find product by id error: %w", err)
	}

	return &product, nil
}
