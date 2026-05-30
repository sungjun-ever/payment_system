package redis

import (
	"payment_system/internal/config"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	redis *redis.Client
}

func NewRedis(cfg *config.Config) *Redis {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost + ":" + cfg.RedisPort,
	})

	return &Redis{
		redis: rdb,
	}
}
