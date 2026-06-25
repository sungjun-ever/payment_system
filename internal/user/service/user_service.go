package service

import (
	"context"
	"fmt"
	"order_system/internal/pkg/hashing"
	"order_system/internal/user/domain"
	"order_system/internal/user/repository"
)

type UserService struct {
	userRepo repository.UserGormRepository
}

func NewUserService(userRepo repository.UserGormRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (us *UserService) CreateUser(ctx context.Context, dto domain.CreateRequest) (*domain.User, error) {
	hashedPassword, err := hashing.HashPassword(dto.Password)

	if err != nil {
		return nil, fmt.Errorf("hash password error: %w", err)
	}

	cUser := &domain.User{
		Name:     dto.Name,
		Email:    dto.Email,
		Password: hashedPassword,
	}

	err = us.userRepo.Create(ctx, cUser)

	if err != nil {
		return nil, fmt.Errorf("create user error: %w", err)
	}

	return cUser, nil
}
