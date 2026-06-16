package registry

import (
	"log/slog"
	"payment_system/internal/auth"
	"payment_system/internal/config"
	"payment_system/internal/idempotency"
	"payment_system/internal/order"
	"payment_system/internal/product"
	"payment_system/internal/user"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Logger             *slog.Logger
	Cfg                *config.Config
	Mysql              *gorm.DB
	Rds                *redis.Client
	UserHandler        *user.UserHandler
	AuthHandler        *auth.AuthHandler
	ProductHandler     *product.ProductHandler
	OrderHandler       *order.OrderHandler
	IdempotencyHandler *idempotency.IdempotencyHandler
}

func NewContainer(
	logger *slog.Logger,
	cfg *config.Config,
	mysql *gorm.DB,
	rds *redis.Client,
) *Container {
	// repo
	userRepo := user.NewUserRepository(mysql)
	authRepo := auth.NewAuthRepository(rds)
	productRepo := product.NewProductRepository(mysql, rds)
	inventoryRepo := product.NewInventoryRepository(mysql, rds)
	orderRepo := order.NewOrderRepository(mysql)
	orderItemRepo := order.NewOrderItemRepository(mysql)
	idempotencyRepo := idempotency.NewIdempotencyKeyRepository(mysql, rds)

	// svc
	userSvc := user.NewUserService(userRepo)
	authSvc := auth.NewAuthService(authRepo, userRepo)
	productSvc := product.NewProductService(logger, productRepo, inventoryRepo)
	idempotencySvc := idempotency.NewIdempotencyService(idempotencyRepo)
	orderSvc := order.NewOrderService(logger, orderRepo, orderItemRepo, idempotencyRepo, inventoryRepo)

	// handler
	userHandler := user.NewUserHandler(userSvc)
	authHandler := auth.NewAuthHandler(*cfg, authSvc)
	productHandler := product.NewProductHandler(productSvc)
	orderHandler := order.NewOrderHandler(orderSvc)
	idempotencyHandler := idempotency.NewIdempotencyHandler(idempotencySvc)

	return &Container{
		Logger:             logger,
		Cfg:                cfg,
		Mysql:              mysql,
		Rds:                rds,
		UserHandler:        userHandler,
		AuthHandler:        authHandler,
		ProductHandler:     productHandler,
		OrderHandler:       orderHandler,
		IdempotencyHandler: idempotencyHandler,
	}
}
