package worker

import (
	"order_system/internal/config"
	"order_system/internal/notification/slack"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	productrepository "order_system/internal/product/repository"
	productworker "order_system/internal/worker"
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
	inventoryJobRepo := productrepository.NewInventoryJobGormRepository(mysql)
	inventoryRedisRepo := productrepository.NewInventoryRedisRepository(rds)

	inventoryRestoreWorker := productworker.NewInventoryRestoreWorker(
		slackSender,
		inventoryJobRepo,
		inventoryRedisRepo,
	)

	return &Container{
		Cfg:                    cfg,
		Mysql:                  mysql,
		Rds:                    rds,
		InventoryRestoreWorker: &inventoryRestoreWorker,
	}
}
