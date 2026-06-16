package rediskey

import "github.com/google/uuid"

const (
	IdempotencyKeyPrefix = "idempotency:"
)

func IdempotencyLockKey(idempotencyKey string) string {
	return "lock:" + IdempotencyKeyPrefix + idempotencyKey
}

func IdempotencyLockToken() string {
	return uuid.New().String()
}
