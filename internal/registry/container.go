package registry

import (
	"log/slog"
	"payment_system/internal/auth"
	"payment_system/internal/auth/repository"
	"payment_system/internal/config"
	"payment_system/internal/idempotency"
	repository4 "payment_system/internal/idempotency/repository"
	"payment_system/internal/order"
	"payment_system/internal/product"
	repository2 "payment_system/internal/product/repository"
	"payment_system/internal/user"
	repository3 "payment_system/internal/user/repository"

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
	userGormRepo := repository3.NewUserGormRepository(mysql)
	authRedisRepo := repository.NewAuthRedisRepository(rds)
	productGormRepo := repository2.NewProductGormRepository(mysql)
	productRedisRepo := repository2.NewProductRedisRepository(rds)
	inventoryGormRepo := repository2.NewInventoryGormRepository(mysql)
	inventoryRedisRepo := repository2.NewInventoryRedisRepository(rds)
	idempotencyGormRepo := repository4.NewIdempotencyGormRepository(mysql)
	idempotencyRedisRepo := repository4.NewIdempotencyRedisRepository(rds)
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
