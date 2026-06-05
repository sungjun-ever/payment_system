package repository

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/model"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = fmt.Errorf("db: user not found")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uint) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
}

type userRepository struct {
	mysql *gorm.DB
}

func NewUserRepository(mysql *gorm.DB) UserRepository {
	return &userRepository{mysql}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	err := gorm.G[model.User](r.mysql).Create(ctx, user)

	if err != nil {
		return fmt.Errorf("db: create user error: %w", err)
	}

	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	user, err := gorm.G[model.User](r.mysql).Where("id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrUserNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find user by id error: %w", err)
	}

	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := gorm.G[model.User](r.mysql).Where("email = ?", email).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrUserNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("db: find user by email error: %w", err)
	}

	return &user, nil
}
