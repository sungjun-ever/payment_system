package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"order_system/internal/config"
	"order_system/internal/database"
	"order_system/internal/pkg/logger"
	"order_system/internal/redis"
	"order_system/internal/registry/api"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config *config.Config
	Router *gin.Engine
}

func NewApp() *App {
	cfg := config.Load()

	mysql := database.NewMysql(cfg)
	rds := redis.NewRedis(cfg)

	appLogger := logger.NewLogger()

	container := api.NewContainer(appLogger, cfg, mysql, rds)

	router := NewRouter(container)

	return &App{
		Config: cfg,
		Router: router,
	}
}

func (app *App) Run() {
	server := http.Server{
		Addr:              listenAddress(app.Config.AppPort),
		Handler:           app.Router.Handler(),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
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

func listenAddress(port string) string {
	if port == "" {
		return ":8080"
	}

	if strings.HasPrefix(port, ":") {
		return port
	}

	return ":" + port
}
