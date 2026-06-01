package handler

import (
	"payment_system/internal/config"
	authDto "payment_system/internal/dto/auth"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/pkg/token"
	"payment_system/internal/service"

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

	tokens, err := a.as.IssueToken(ctx, a.cfg, user)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.SetCookie("refresh_token",
		tokens.RefreshToken,
		int(token.RefreshDuration.Seconds()),
		"/",
		"",
		false,
		true,
	)

	response.ToSuccessResponse(c, 200, authDto.NewResource(tokens.RefreshToken))
}

func (a *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("userID")

	err := a.as.DeleteToken(ctx, userID.(uint))

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.ToSuccessResponse(c, 200, nil)
}

func (a *AuthHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()
	cookieRefreshToken, _ := c.Request.Cookie("refresh_token")

	if cookieRefreshToken == nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 401, apperr.A003, "token expired", nil))
		return
	}

	userID, _ := c.Get("userID")
	email, _ := c.Get("email")

	tokens, err := a.as.RefreshAccessToken(ctx, a.cfg, cookieRefreshToken.Value, userID.(uint), email.(string))

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.SetCookie("refresh_token",
		tokens.RefreshToken,
		int(token.RefreshDuration.Seconds()),
		"/",
		"",
		false,
		true,
	)

	response.ToSuccessResponse(c, 200, authDto.NewResource(tokens.AccessToken))
}
