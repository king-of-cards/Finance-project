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

		}
	}

	return r
}
