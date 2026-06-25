package repository

import (
	"context"
	"errors"
	"fmt"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/product/domain"

	"gorm.io/gorm"
)

type ProductGormRepository struct {
	mysql *gorm.DB
}

func NewProductGormRepository(db *gorm.DB) ProductGormRepository {
	return ProductGormRepository{db}
}

func (p *ProductGormRepository) WithTx(tx *gorm.DB) ProductGormRepository {
	return ProductGormRepository{tx}
}

func (p *ProductGormRepository) Transaction(txFn func(tx *gorm.DB) error) error {
	return p.mysql.Transaction(txFn)
}

func (p *ProductGormRepository) Store(ctx context.Context, product *domain.Product) error {
	result := p.mysql.WithContext(ctx).Model(&domain.Product{}).Create(product)

	if result.Error != nil {
		return fmt.Errorf("db: create product error: %w", result.Error)
	}

	return nil
}

func (p *ProductGormRepository) Update(ctx context.Context, id uint, fields *domain.Product) (*domain.Product, error) {
	result := p.mysql.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", id).Updates(fields)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product not found: %w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: update product error: %w", result.Error)
	}

	product, err := p.Find(ctx, id)

	if err != nil {
		return nil, err
	}

	return product, nil
}

func (p *ProductGormRepository) Find(ctx context.Context, id uint) (*domain.Product, error) {
	var product domain.Product

	result := p.mysql.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", id).First(&product)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product find error: %w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: find product by id error: %w", result.Error)
	}

	return &product, nil
}

func (p *ProductGormRepository) Delete(ctx context.Context, id uint) error {
	result := p.mysql.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", id).Delete(ctx)

	if result.Error != nil {
		return fmt.Errorf("db: delete product error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("deleted product not found: %w", dberr.ErrNotFound)
	}

	return nil
}
