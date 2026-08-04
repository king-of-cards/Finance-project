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

	api := r.Group("/api")
	{
		api.POST("/login", authHandler.Login)

		protected := api.Group("/")
		protected.Use(middleware.RequireAuth(pool, jwtSecret))
		{
			protected.GET("/users", userHandler.GetAllUsers)
			protected.POST("/logout", authHandler.Logout)
		}
	}

	return r
}