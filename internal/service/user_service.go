package service

import (
	"context"
	"payment_system/internal/common/errUtils"
	"payment_system/internal/dto/user"
	"payment_system/internal/model"
	"payment_system/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (us *UserService) CreateUser(ctx context.Context, dto user.CreateRequest) (*model.User, error) {
	cUser := &model.User{
		Name:  dto.Name,
		Email: dto.Email,
	}

	err := us.userRepo.Create(ctx, cUser)

	if err != nil {
		return nil, errUtils.NewApiError(errUtils.LevelError, 500, errUtils.S001, "create_user db error", nil)
	}

	return cUser, nil
}
