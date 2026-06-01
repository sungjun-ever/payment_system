package registry

import (
	"log/slog"
	"payment_system/internal/handler"
	"payment_system/internal/repository"
	"payment_system/internal/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Logger      *slog.Logger
	Mysql       *gorm.DB
	Rds         *redis.Client
	UserHandler *handler.UserHandler
	AuthHandler *handler.AuthHandler
}

func NewContainer(
	logger *slog.Logger,
	mysql *gorm.DB,
	rds *redis.Client,
) *Container {
	// repo
	userRepo := repository.NewUserRepository(mysql)
	authRepo := repository.NewAuthRepository(rds)

	// svc
	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(authRepo, userRepo)

	// handler
	userHandler := handler.NewUserHandler(userSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	return &Container{
		Logger:      logger,
		Mysql:       mysql,
		Rds:         rds,
		UserHandler: userHandler,
		AuthHandler: authHandler,
	}
}
