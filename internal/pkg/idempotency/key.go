package idempotency

import (
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrCreateIdempotencyKeyErr = fmt.Errorf("create idempotency key error")
)

func GenerateKey() (string, error) {
	key, err := uuid.NewV7()

	if err != nil {
		return "", ErrCreateIdempotencyKeyErr
	}

	return "idempotency_" + key.String(), nil
}
