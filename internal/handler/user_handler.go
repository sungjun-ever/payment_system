package handler

import (
	userDto "payment_system/internal/dto/user"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/service"

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
	var req userDto.CreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, "유효하지 않은 입력값", nil))
		return
	}

	createdUser, err := u.us.CreateUser(ctx, req)

	if err != nil {
		_ = c.Error(err)
		return
	}

	response.ToSuccessResponse(c, 201, userDto.NewResource(createdUser))
}
