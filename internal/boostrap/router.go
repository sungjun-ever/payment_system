package boostrap

import (
	"order_system/internal/middleware"
	"order_system/internal/registry"

	"github.com/gin-gonic/gin"
)

type Router struct{}

func NewRouter(ct *registry.Container) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.RequestTraceMiddleware(ct.Logger))
	r.Use(middleware.ErrorHandlerMiddleware(ct.Logger))

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

			idempotency := v1.Group("/idempotency")
			{
				idempotency.POST("", ct.IdempotencyHandler.Create)
			}

			products := v1.Group("/products")
			{
				products.POST("", ct.ProductHandler.Create)
				products.GET("/:productID", ct.ProductHandler.Get)
				products.PUT("/:productID", ct.ProductHandler.Update)
				products.DELETE("/:productID", ct.ProductHandler.Delete)
			}

			orders := v1.Group("/orders")
			{
				orders.POST("",
					middleware.IdempotencyKeyMiddleware(),
					middleware.HashRequestBodyMiddleware(1<<20),
					ct.OrderHandler.Create,
				)
				orders.DELETE("/:orderID", ct.OrderHandler.Cancel)

			}

			payments := v1.Group("/payments")
			{
				payments.POST("",
					middleware.IdempotencyKeyMiddleware(),
					middleware.HashRequestBodyMiddleware(1<<20),
					ct.PaymentHandler.Create,
				)
			}
		}

	}

	return r
}
