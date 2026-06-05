package user

import (
	"context"
	"fmt"
	"payment_system/internal/pkg/hashing"
)

type UserService struct {
	userRepo UserRepository
}

func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (us *UserService) CreateUser(ctx context.Context, dto CreateRequest) (*User, error) {
	hashedPassword, err := hashing.HashPassword(dto.Password)

	if err != nil {
		return nil, fmt.Errorf("hash password error: %w", err)
	}

	cUser := &User{
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
