package boostrap

import (
	"payment_system/internal/common/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct{}

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.ErrorLogMiddleware())

	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			v1.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "pong"})
			})
		}
	}

	return r
}
