package registry

import (
	"log/slog"
	"payment_system/internal/config"
	"payment_system/internal/handler"
	"payment_system/internal/repository"
	"payment_system/internal/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Logger         *slog.Logger
	Cfg            *config.Config
	Mysql          *gorm.DB
	Rds            *redis.Client
	UserHandler    *handler.UserHandler
	AuthHandler    *handler.AuthHandler
	ProductHandler *handler.ProductHandler
}

func NewContainer(
	logger *slog.Logger,
	cfg *config.Config,
	mysql *gorm.DB,
	rds *redis.Client,
) *Container {
	// repo
	userRepo := repository.NewUserRepository(mysql)
	authRepo := repository.NewAuthRepository(rds)
	productRepo := repository.NewProductRepository(mysql)
	inventoryRepo := repository.NewInventoryRepository(mysql)

	// svc
	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(authRepo, userRepo)
	productSvc := service.NewProductService(productRepo, inventoryRepo)

	// handler
	userHandler := handler.NewUserHandler(userSvc)
	authHandler := handler.NewAuthHandler(*cfg, authSvc)
	productHandler := handler.NewProductHandler(productSvc)

	return &Container{
		Logger:         logger,
		Cfg:            cfg,
		Mysql:          mysql,
		Rds:            rds,
		UserHandler:    userHandler,
		AuthHandler:    authHandler,
		ProductHandler: productHandler,
	}
}
