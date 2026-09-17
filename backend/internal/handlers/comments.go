package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type CommentHandler struct {
	Pool *pgxpool.Pool
}

func NewCommentHandler(pool *pgxpool.Pool) *CommentHandler {
	return &CommentHandler{Pool: pool}
}

func (h *CommentHandler) GetComments(c *gin.Context) {
	poNumber := c.Param("id")

	comments, err := db.GetComments(c.Request.Context(), h.Pool, poNumber)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

type addCommentRequest struct {
	CommentText string `json:"comment_text" binding:"required"`
}

func (h *CommentHandler) AddComment(c *gin.Context) {
	poNumber := c.Param("id")

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	authorID := userIDVal.(string)

	var req addCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_text is required"})
		return
	}

	comment, err := db.AddComment(c.Request.Context(), h.Pool, poNumber, authorID, req.CommentText)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add comment"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	idParam := c.Param("id")
	commentID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	err = db.DeleteComment(c.Request.Context(), h.Pool, commentID)
	if err != nil {
		if err == db.ErrCommentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}
