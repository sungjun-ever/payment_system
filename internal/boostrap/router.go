package boostrap

import (
	"payment_system/internal/common/middleware"
	"payment_system/internal/registry"

	"github.com/gin-gonic/gin"
)

type Router struct{}

func NewRouter(ct *registry.Container) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.RequestTraceMiddleware())
	r.Use(middleware.RequestLoggerMiddleware(ct.Logger))
	r.Use(middleware.ErrorLogMiddleware())

	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			v1.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "pong"})
			})
		}

		users := v1.Group("/users")
		{
			users.POST("", ct.UserHandler.Create)
		}
	}

	return r
}
