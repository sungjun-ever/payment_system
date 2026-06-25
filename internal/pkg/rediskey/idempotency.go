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

func IdempotencyStatus(idempotencyKey string) string {
	return IdempotencyKeyPrefix + idempotencyKey + ":status"
}
