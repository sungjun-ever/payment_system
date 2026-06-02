package handler

import (
	"errors"
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
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	user, err := a.as.ValidUser(ctx, req)

	if err != nil {
		_ = c.Error(toAppError(err))
		return
	}

	tokens, err := a.as.IssueToken(ctx, a.cfg, user)

	if err != nil {
		_ = c.Error(toAppError(err))
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

	accessToken, _ := c.Get("access_token")
	claims, _ := c.Get("accessClaims")

	err := a.as.DeleteToken(ctx, accessToken.(string), claims.(*token.AccessClaims))

	if err != nil {
		_ = c.Error(toAppError(err))
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

	claims, _ := c.Get("accessClaims")

	tokens, err := a.as.RotateToken(ctx, a.cfg, cookieRefreshToken.Value, claims.(*token.AccessClaims))

	if err != nil {
		_ = c.Error(toAppError(err))
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

func toAppError(err error) *apperr.AppError {
	switch {
	case errors.Is(err, service.ErrInvalidToken):
		return apperr.NewAppError(apperr.LevelWarn, 401, apperr.A002, err, nil)

	case errors.Is(err, service.ErrTokenNotFound):
		return apperr.NewAppError(apperr.LevelInfo, 401, apperr.A003, err, nil)

	case errors.Is(err, service.ErrInvalidCredentials):
		return apperr.NewAppError(apperr.LevelInfo, 401, apperr.C001, err, nil)

	case errors.Is(err, service.ErrTokenConflict):
		return apperr.NewAppError(apperr.LevelWarn, 409, apperr.A002, err, nil)

	default:
		return apperr.NewAppError(apperr.LevelError, 500, apperr.S001, err, nil)
	}

}
