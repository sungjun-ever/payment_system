package worker

import (
	"context"
	"log/slog"
	"order_system/internal/config"
	"order_system/internal/database"
	"order_system/internal/redis"
	"order_system/internal/registry/worker"
	"os/signal"
	"syscall"
)

type App struct {
	container *worker.Container
}

func NewApp() *App {
	cfg := config.Load()

	mysql := database.NewMysql(cfg)
	rds := redis.NewRedis(cfg)

	container := worker.NewContainer(cfg, mysql, rds)

	return &App{
		container,
	}
}

func (app *App) Run() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	defer stop()

	slog.InfoContext(ctx, "inventory restore worker started")
	app.container.InventoryRestoreWorker.Start(ctx)
	slog.InfoContext(context.Background(), "inventory restore worker stopped")
}
