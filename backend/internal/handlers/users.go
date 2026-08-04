package handlers 

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/king-of-cards/finance-project/internal/db"

)

type UserHandler struct {
	Pool *pgxpool.Pool
}

func NewUserHandler(pool *pgxpool.Pool) *UserHandler {
	return &UserHandler{Pool: pool}
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	users,err := db.GetAllUsers(c.Request.Context(),h.Pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return 
	}

	c.JSON(http.StatusOK, gin.H{"users": users})


}