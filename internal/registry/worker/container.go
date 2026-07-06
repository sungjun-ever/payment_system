package worker

import (
	"order_system/internal/config"
	"order_system/internal/notification/slack"
	orderrepository "order_system/internal/order/repository"
	productworker "order_system/internal/worker/product"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	productrepository "order_system/internal/product/repository"
)

type Container struct {
	Cfg                    *config.Config
	Mysql                  *gorm.DB
	Rds                    *redis.Client
	InventoryRestoreWorker *productworker.InventoryRestoreWorker
}

func NewContainer(
	cfg *config.Config,
	mysql *gorm.DB,
	rds *redis.Client,
) *Container {
	slackClient := slack.NewSlackClient(cfg.SlackWebhookURL)
	slackSender := slack.NewSender(slackClient)

	// repository
	inventoryJobGormRepo := productrepository.NewInventoryJobGormRepository(mysql)
	inventoryRedisRepo := productrepository.NewInventoryRedisRepository(rds)
	inventoryGormRepo := productrepository.NewInventoryGormRepository(mysql)
	orderItemRepo := orderrepository.NewOrderItemGormRepository(mysql)
	productStore := productworker.NewProductStore(mysql)

	inventoryRestoreWorker := productworker.NewInventoryRestoreWorker(
		slackSender,
		inventoryJobGormRepo,
		inventoryRedisRepo,
		inventoryGormRepo,
		orderItemRepo,
		productStore,
	)

	return &Container{
		Cfg:                    cfg,
		Mysql:                  mysql,
		Rds:                    rds,
		InventoryRestoreWorker: &inventoryRestoreWorker,
	}
}
