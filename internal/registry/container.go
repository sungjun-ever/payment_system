package registry

import (
	"log/slog"
	"payment_system/internal/handler"
	"payment_system/internal/redis"
	"payment_system/internal/repository"
	"payment_system/internal/service"

	"gorm.io/gorm"
)

type Container struct {
	Logger      *slog.Logger
	Mysql       *gorm.DB
	Rds         *redis.Redis
	UserHandler *handler.UserHandler
}

func NewContainer(
	logger *slog.Logger,
	mysql *gorm.DB,
	rds *redis.Redis,
) *Container {
	// repo
	userRepo := repository.NewUserRepository(mysql)

	// svc
	userSvc := service.NewUserService(userRepo)

	// handler
	userHandler := handler.NewUserHandler(userSvc)

	return &Container{
		Logger:      logger,
		Mysql:       mysql,
		Rds:         rds,
		UserHandler: userHandler,
	}
}
