package registry

import (
	"log/slog"
	"payment_system/internal/database"
	"payment_system/internal/redis"
)

type Container struct {
	Logger *slog.Logger
	Mysql  *database.Mysql
	Rds    *redis.Redis
}

func NewContainer(
	logger *slog.Logger,
	mysql *database.Mysql,
	rds *redis.Redis,
) *Container {
	return &Container{
		Logger: logger,
		Mysql:  mysql,
		Rds:    rds,
	}
}
