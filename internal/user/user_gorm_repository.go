package user

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/pkg/apperr/dberr"

	"gorm.io/gorm"
)

type UserGormRepository struct {
	mysql *gorm.DB
}

func NewUserGormRepository(mysql *gorm.DB) UserGormRepository {
	return UserGormRepository{mysql}
}

func (r *UserGormRepository) Create(ctx context.Context, user *User) error {
	result := r.mysql.WithContext(ctx).Model(&User{}).Create(user)

	if result.Error != nil {
		return fmt.Errorf("db: create user error: %w", result.Error)
	}

	return nil
}

func (r *UserGormRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	var user User
	result := r.mysql.WithContext(ctx).Model(&user).Where("id = ?", id).First(ctx)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: find user by id error: %w", result.Error)
	}

	return &user, nil
}

func (r *UserGormRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	result := r.mysql.WithContext(ctx).Model(&user).Where("email = ?", email).First(ctx)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w", dberr.ErrNotFound)
		}
		return nil, fmt.Errorf("db: find user by email error: %w", result.Error)
	}

	return &user, nil
}
