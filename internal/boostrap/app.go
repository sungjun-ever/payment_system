package boostrap

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"payment_system/internal/common/logger"
	"payment_system/internal/config"
	"payment_system/internal/database"
	"payment_system/internal/redis"
	"payment_system/internal/registry"
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

	container := registry.NewContainer(appLogger, mysql, rds)

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
