package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var (
	ErrAccessTokenExpired = errors.New("access token expired")
	ErrInvalidAccessToken = errors.New("invalid token")
)

func NewAccessClaims(id uint, email string) *AccessClaims {
	return &AccessClaims{
		id,
		email,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
}

func (a *AccessClaims) CreateAccessToken(secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, a)
	return token.SignedString([]byte(secret))
}

func ParseValidAccessToken(secret string, tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&AccessClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, ErrInvalidAccessToken

		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, ErrInvalidAccessToken

		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrAccessTokenExpired

		default:
			return nil, ErrInvalidAccessToken
		}
	}

	if !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	parsedClaims, ok := token.Claims.(*AccessClaims)
	if !ok {
		return nil, ErrInvalidAccessToken
	}

	return parsedClaims, nil
}
