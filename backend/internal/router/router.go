package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/handlers"
	"github.com/king-of-cards/finance-project/internal/middleware"
)

func New(pool *pgxpool.Pool, jwtSecret string, jwtExpiryHr int) *gin.Engine {
	r := gin.Default()

	userHandler := handlers.NewUserHandler(pool)
	authHandler := handlers.NewAuthHandler(pool, jwtSecret, jwtExpiryHr)
	vendorHandler := handlers.NewVendorHandler(pool)
	poHandler := handlers.NewPurchaseOrderHandler(pool)

	api := r.Group("/api")
	{
		api.POST("/login", authHandler.Login)

		protected := api.Group("/")
		protected.Use(middleware.RequireAuth(pool, jwtSecret))
		{
			protected.GET("/users", userHandler.GetAllUsers)
			protected.POST("/logout", authHandler.Logout)
			protected.GET("/auth/me", authHandler.Me)
			protected.GET("/finance-users", userHandler.GetFinanceUsers)
			protected.GET("/vendors", vendorHandler.GetVendors)
			protected.GET("/vendors/:id", vendorHandler.GetVendorDetail)
			protected.GET("/vendors/:id/orders", vendorHandler.GetVendorOrders)
			protected.POST("/vendors", vendorHandler.CreateVendor)
			protected.PATCH("/vendors/:id", vendorHandler.UpdateVendor)
			protected.DELETE("/vendors/:id", vendorHandler.DeactivateVendor)
			protected.GET("/purchase-orders", poHandler.GetPurchaseOrders)
			protected.GET("/purchase-orders/:id", poHandler.GetPurchaseOrderDetail)
			protected.POST("/purchase-orders", poHandler.CreatePurchaseOrder)
			protected.GET("/purchase-orders/export", poHandler.ExportPurchaseOrders)

		}
	}

	return r
}
