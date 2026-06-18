package handler

import (
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/user/domain"
	"payment_system/internal/user/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	us *service.UserService
}

func NewUserHandler(us *service.UserService) *UserHandler {
	return &UserHandler{us}
}

func (u *UserHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()
	var req domain.CreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelInfo, 400, apperr.C001, err, nil))
		return
	}

	createdUser, err := u.us.CreateUser(ctx, req)

	if err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 500, apperr.S001, err, nil))
		return
	}

	response.ToSuccessResponse(c, 201, domain.NewResource(createdUser))
}
