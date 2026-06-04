package boostrap

import (
	"payment_system/internal/pkg/middleware"
	"payment_system/internal/registry"

	"github.com/gin-gonic/gin"
)

type Router struct{}

func NewRouter(ct *registry.Container) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.RequestTraceMiddleware(ct.Logger))
	r.Use(middleware.ErrorLogMiddleware(ct.Logger))

	api := r.Group("/api")
	{
		api.POST("/v1/auth/login", ct.AuthHandler.Login)
		api.POST("/v1/auth/refresh", ct.AuthHandler.Refresh)
		api.POST("/v1/users", ct.UserHandler.Create)

		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware(ct.Rds, ct.Cfg))
		v1 := authorized.Group("/v1")
		{
			auth := v1.Group("/auth")
			{
				auth.DELETE("/logout", ct.AuthHandler.Logout)
			}

			//users := v1.Group("/users"){}

			products := v1.Group("/products")
			{
				products.POST("", ct.ProductHandler.Create)
				products.GET("/:productID", ct.ProductHandler.Get)
			}
		}

	}

	return r
}
