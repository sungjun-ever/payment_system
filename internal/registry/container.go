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
	userGormRepo := user.NewUserGormRepository(mysql)
	authRedisRepo := auth.NewAuthRedisRepository(rds)
	productGormRepo := product.NewProductGormRepository(mysql)
	productRedisRepo := product.NewProductRedisRepository(rds)
	inventoryGormRepo := product.NewInventoryGormRepository(mysql)
	inventoryRedisRepo := product.NewInventoryRedisRepository(rds)
	idempotencyGormRepo := idempotency.NewIdempotencyGormRepository(mysql)
	idempotencyRedisRepo := idempotency.NewIdempotencyRedisRepository(rds)
	orderUow := order.NewOrderUnitOfWork(mysql, idempotencyGormRepo)

	// svc
	userSvc := user.NewUserService(userGormRepo)
	authSvc := auth.NewAuthService(authRedisRepo, userGormRepo)
	productSvc := product.NewProductService(logger, productGormRepo, productRedisRepo, inventoryGormRepo, inventoryRedisRepo)
	idempotencySvc := idempotency.NewIdempotencyService(idempotencyGormRepo)
	orderSvc := order.NewOrderService(logger, orderUow, idempotencyGormRepo, idempotencyRedisRepo, inventoryGormRepo, inventoryRedisRepo)

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
