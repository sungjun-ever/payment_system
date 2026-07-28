package api

import (
	"log/slog"
	"order_system/internal/config"
	idempotencyhandler "order_system/internal/idempotency/handler"
	idempotencyrepository "order_system/internal/idempotency/repository"
	idempotencyservice "order_system/internal/idempotency/service"
	"order_system/internal/notification/slack"
	orderhandler "order_system/internal/order/handler"
	orderrepository "order_system/internal/order/repository"
	orderservice "order_system/internal/order/service"
	paymenthandler "order_system/internal/payment/handler"
	paymentrepository "order_system/internal/payment/repository"
	paymentservice "order_system/internal/payment/service"
	"order_system/internal/pkg/pg/toss"
	producthandler "order_system/internal/product/handler"
	productRepository "order_system/internal/product/repository"
	productservice "order_system/internal/product/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Logger             *slog.Logger
	Cfg                *config.Config
	Mysql              *gorm.DB
	Rds                *redis.Client
	ProductHandler     *producthandler.ProductHandler
	OrderHandler       *orderhandler.OrderHandler
	IdempotencyHandler *idempotencyhandler.IdempotencyHandler
	PaymentHandler     *paymenthandler.PaymentHandler
}

func NewContainer(
	logger *slog.Logger,
	cfg *config.Config,
	mysql *gorm.DB,
	rds *redis.Client,
) *Container {
	slackClient := slack.NewSlackClient(cfg.SlackWebhookURL)
	slackSender := slack.NewSender(slackClient)

	tossProvider := toss.NewTossProvider(cfg.TossSecretKey)

	// repo
	productGormRepo := productRepository.NewProductGormRepository(mysql)
	productRedisRepo := productRepository.NewProductRedisRepository(rds)
	inventoryGormRepo := productRepository.NewInventoryGormRepository(mysql)
	inventoryRedisRepo := productRepository.NewInventoryRedisRepository(rds)
	idempotencyGormRepo := idempotencyrepository.NewIdempotencyGormRepository(mysql)
	idempotencyRedisRepo := idempotencyrepository.NewIdempotencyRedisRepository(rds)
	orderItemRepo := orderrepository.NewOrderItemGormRepository(mysql)
	orderStore := orderrepository.NewOrderUnitOfWork(
		mysql,
		idempotencyGormRepo,
		productGormRepo,
		inventoryGormRepo,
		orderItemRepo,
	)
	orderRepo := orderrepository.NewOrderGormRepository(mysql)
	paymentRepo := paymentrepository.NewPaymentGormRepository(mysql)
	attemptRepo := paymentrepository.NewAttemptGormRepository(mysql)
	paymentStore := paymentrepository.NewPaymentStore(
		mysql,
		rds,
		paymentRepo,
		attemptRepo,
		orderRepo,
		idempotencyGormRepo,
	)

	// svc
	productSvc := productservice.NewProductService(
		logger,
		productGormRepo,
		&productRedisRepo,
		inventoryGormRepo,
		&inventoryRedisRepo,
	)
	idempotencySvc := idempotencyservice.NewIdempotencyService(idempotencyGormRepo)
	orderSvc := orderservice.NewOrderService(
		logger,
		orderStore,
		&idempotencyRedisRepo,
		&inventoryRedisRepo,
		slackSender,
	)
	paymentSvc := paymentservice.NewPaymentService(
		logger,
		paymentStore,
		&paymentRepo,
		&idempotencyRedisRepo,
		slackSender,
		tossProvider,
	)

	productHandler := producthandler.NewProductHandler(productSvc)
	orderHandler := orderhandler.NewOrderHandler(orderSvc)
	idempotencyHandler := idempotencyhandler.NewIdempotencyHandler(idempotencySvc)
	paymentHandler := paymenthandler.NewPaymentHandler(paymentSvc)

	return &Container{
		Logger:             logger,
		Cfg:                cfg,
		Mysql:              mysql,
		Rds:                rds,
		ProductHandler:     productHandler,
		OrderHandler:       orderHandler,
		IdempotencyHandler: idempotencyHandler,
		PaymentHandler:     paymentHandler,
	}
}
