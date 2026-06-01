package repository

import (
	"context"
	"payment_system/internal/model"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
}

type userRepository struct {
	mysql *gorm.DB
}

func NewUserRepository(mysql *gorm.DB) UserRepository {
	return &userRepository{mysql}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return gorm.G[model.User](r.mysql).Create(ctx, user)
}
