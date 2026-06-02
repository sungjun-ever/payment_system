package service

import (
	"context"
	"fmt"
	userDto "payment_system/internal/dto/user"
	"payment_system/internal/model"
	"payment_system/internal/pkg/hashing"
	"payment_system/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (us *UserService) CreateUser(ctx context.Context, dto userDto.CreateRequest) (*model.User, error) {
	hashedPassword, err := hashing.HashPassword(dto.Password)

	if err != nil {
		return nil, fmt.Errorf("hash password error: %w", err)
	}

	cUser := &model.User{
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
