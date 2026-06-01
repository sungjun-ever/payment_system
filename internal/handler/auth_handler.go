package handler

import (
	"payment_system/internal/config"
	authDto "payment_system/internal/dto/auth"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg config.Config
	as  service.AuthService
}

func NewAuthHandler(cfg config.Config, as service.AuthService) *AuthHandler {
	return &AuthHandler{cfg, as}
}

func (a *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req authDto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, "유효하지 않은 입력값", nil))
		return
	}

	user, err := a.as.ValidUser(ctx, req)

	if err != nil {
		_ = c.Error(err)
		return
	}

	refreshDuration := 24 * 7 * time.Hour

	tokens, err := a.as.IssueToken(ctx, a.cfg, user, refreshDuration)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.SetCookie("refresh_token",
		tokens.RefreshToken,
		int(refreshDuration.Seconds()),
		"/",
		"",
		false,
		true,
	)

	response.ToSuccessResponse(c, 200, authDto.NewResource(tokens.RefreshToken))
}
