package registry

import (
	"log/slog"
	authhandler "payment_system/internal/auth/handler"
	authrepository "payment_system/internal/auth/repository"
	authservice "payment_system/internal/auth/service"
	"payment_system/internal/config"
	idempotencyhandler "payment_system/internal/idempotency/handler"
	idempotencyrepository "payment_system/internal/idempotency/repository"
	idempotencyservice "payment_system/internal/idempotency/service"
	"payment_system/internal/notification/slack"
	orderhandler "payment_system/internal/order/handler"
	orderrepository "payment_system/internal/order/repository"
	orderservice "payment_system/internal/order/service"
	producthandler "payment_system/internal/product/handler"
	productRepository "payment_system/internal/product/repository"
	productservice "payment_system/internal/product/service"
	userhandler "payment_system/internal/user/handler"
	userrepository "payment_system/internal/user/repository"
	userservice "payment_system/internal/user/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Logger             *slog.Logger
	Cfg                *config.Config
	Mysql              *gorm.DB
	Rds                *redis.Client
	UserHandler        *userhandler.UserHandler
	AuthHandler        *authhandler.AuthHandler
	ProductHandler     *producthandler.ProductHandler
	OrderHandler       *orderhandler.OrderHandler
	IdempotencyHandler *idempotencyhandler.IdempotencyHandler
}

func NewContainer(
	logger *slog.Logger,
	cfg *config.Config,
	mysql *gorm.DB,
	rds *redis.Client,
) *Container {
	//emailClient := email.NewEmailClient("test@test.com", "test_admin")
	//emailSender := email.NewSender(emailClient)

	slackClient := slack.NewSlackClient(cfg.SlackWebhookURL)
	slackSender := slack.NewSender(slackClient)

	// repo
	userGormRepo := userrepository.NewUserGormRepository(mysql)
	authRedisRepo := authrepository.NewAuthRedisRepository(rds)
	productGormRepo := productRepository.NewProductGormRepository(mysql)
	productRedisRepo := productRepository.NewProductRedisRepository(rds)
	inventoryGormRepo := productRepository.NewInventoryGormRepository(mysql)
	inventoryRedisRepo := productRepository.NewInventoryRedisRepository(rds)
	idempotencyGormRepo := idempotencyrepository.NewIdempotencyGormRepository(mysql)
	idempotencyRedisRepo := idempotencyrepository.NewIdempotencyRedisRepository(rds)
	orderUow := orderrepository.NewOrderUnitOfWork(mysql, idempotencyGormRepo)

	// svc
	userSvc := userservice.NewUserService(userGormRepo)
	authSvc := authservice.NewAuthService(authRedisRepo, userGormRepo)
	productSvc := productservice.NewProductService(
		logger,
		productGormRepo,
		productRedisRepo,
		inventoryGormRepo,
		inventoryRedisRepo,
	)
	idempotencySvc := idempotencyservice.NewIdempotencyService(idempotencyGormRepo)
	orderSvc := orderservice.NewOrderService(
		logger,
		orderUow,
		idempotencyGormRepo,
		idempotencyRedisRepo,
		inventoryGormRepo,
		inventoryRedisRepo,
		slackSender,
	)

	// orderhandler
	userHandler := userhandler.NewUserHandler(userSvc)
	authHandler := authhandler.NewAuthHandler(*cfg, authSvc)
	productHandler := producthandler.NewProductHandler(productSvc)
	orderHandler := orderhandler.NewOrderHandler(orderSvc)
	idempotencyHandler := idempotencyhandler.NewIdempotencyHandler(idempotencySvc)

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
