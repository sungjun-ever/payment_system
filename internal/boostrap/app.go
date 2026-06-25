package boostrap

import (
	"context"
	"errors"
	"log"
	"net/http"
	"order_system/internal/config"
	"order_system/internal/database"
	"order_system/internal/pkg/logger"
	"order_system/internal/redis"
	"order_system/internal/registry"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type App struct {
	Router *gin.Engine
}

func NewApp() *App {
	cfg := config.Load()

	mysql := database.NewMysql(cfg)
	rds := redis.NewRedis(cfg)

	appLogger := logger.NewLogger()

	container := registry.NewContainer(appLogger, cfg, mysql, rds)

	router := NewRouter(container)

	return &App{
		Router: router,
	}
}

func (app *App) Run() {
	server := http.Server{
		Addr:    ":8080",
		Handler: app.Router.Handler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	log.Println("Server exiting")
}
