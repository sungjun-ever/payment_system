package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type RefreshClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func NewRefreshClaims(id uint) *RefreshClaims {
	return &RefreshClaims{
		id,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
}

func (r *RefreshClaims) CreateRefreshToken(secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, r)
	return token.SignedString([]byte(secret))
}
