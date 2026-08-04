package handlers 

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/auth"
	"github.com/king-of-cards/finance-project/internal/db"

)

type AuthHandler struct {
	Pool        *pgxpool.Pool
	JWTSecret   string
	JWTExpiryHr int
}

func NewAuthHandler(pool *pgxpool.Pool, jwtSecret string, jwtExpiryHr int) *AuthHandler {
	return &AuthHandler{Pool: pool, JWTSecret: jwtSecret, JWTExpiryHr: jwtExpiryHr}
}

type LoginRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and password are required"})
		return
	}
	user, err := db.GetUserByID(c.Request.Context(), h.Pool, req.UserID)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, expiresAt, _, err := auth.GenerateToken(user.UserID, user.Role, h.JWTSecret, h.JWTExpiryHr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"user": gin.H{
			"user_id": user.UserID,
			"name":    user.Name,
			"role":    user.Role,
		},
	})


}


func (h *AuthHandler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization header"})
		return 
	}
	tokenStr := strings.TrimPrefix(header,"Bearer ")

	claims, err := auth.ValidateToken(tokenStr, h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	err = db.RevokeToken(c.Request.Context(),h.Pool, claims.ID, claims.ExpiresAt.Time )

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
	
}