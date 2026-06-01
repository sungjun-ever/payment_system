package redis

import (
	"payment_system/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg *config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost + ":" + cfg.RedisPort,
	})

	return rdb
}
