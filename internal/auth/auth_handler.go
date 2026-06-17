package auth

import (
	"errors"
	"payment_system/internal/auth/domain"

	"payment_system/internal/config"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	"payment_system/internal/pkg/token"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg config.Config
	as  AuthService
}

func NewAuthHandler(cfg config.Config, as AuthService) *AuthHandler {
	return &AuthHandler{cfg, as}
}

func (a *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	user, err := a.as.ValidUser(ctx, req)

	if err != nil {
		_ = c.Error(ToAppError(err))
		return
	}

	tokens, err := a.as.IssueToken(ctx, a.cfg, user)

	if err != nil {
		_ = c.Error(ToAppError(err))
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

	response.ToSuccessResponse(c, 200, domain.NewResource(tokens.AccessToken))
}

func (a *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	accessToken, _ := c.Get("accessToken")
	claims, _ := c.Get("accessClaims")

	err := a.as.DeleteToken(ctx, accessToken.(string), claims.(*token.AccessClaims))

	if err != nil {
		_ = c.Error(ToAppError(err))
		return
	}

	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.ToSuccessResponse(c, 200, nil)
}

func (a *AuthHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()
	cookieRefreshToken, _ := c.Request.Cookie("refresh_token")

	if cookieRefreshToken == nil {
		_ = c.Error(apperr.NewAppError(
			apperr.LevelError,
			401,
			apperr.A003,
			errors.New("cookie refresh token not exists"),
			nil,
		))
		return
	}

	tokens, err := a.as.RotateToken(ctx, a.cfg, cookieRefreshToken.Value)

	if err != nil {
		_ = c.Error(ToAppError(err))
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

	response.ToSuccessResponse(c, 200, domain.NewResource(tokens.AccessToken))
}
