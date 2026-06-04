package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type RefreshClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

var (
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

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

func ParseValidRefreshToken(secret string, tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&RefreshClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, ErrInvalidRefreshToken

		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, ErrInvalidRefreshToken

		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrRefreshTokenExpired

		default:
			return nil, ErrInvalidRefreshToken
		}
	}

	if !token.Valid {
		return nil, ErrInvalidRefreshToken
	}

	parsedClaims, ok := token.Claims.(*RefreshClaims)
	if !ok {
		return nil, ErrInvalidRefreshToken
	}

	return parsedClaims, nil
}
