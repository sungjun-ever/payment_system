package token

import (
	"payment_system/internal/pkg/apperr"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewAccessClaims(id uint, email string) *AccessClaims {
	return &AccessClaims{
		id,
		email,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
}

func (a *AccessClaims) ValidAccessToken(secret string, tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, a, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperr.NewAppError(apperr.LevelError, 401, apperr.A002, "invalid token", nil)
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil, apperr.NewAppError(apperr.LevelError, 401, apperr.A002, "parsed token error", nil)
	}

	if claims, ok := token.Claims.(*AccessClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, apperr.NewAppError(apperr.LevelError, 401, apperr.A001, "invalid token", nil)
}

func (a *AccessClaims) CreateAccessToken(secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, a)
	return token.SignedString([]byte(secret))
}
