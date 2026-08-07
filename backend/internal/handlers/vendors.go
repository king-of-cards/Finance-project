package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type VendorHandler struct {
	Pool *pgxpool.Pool
}

func NewVendorHandler(pool *pgxpool.Pool) *VendorHandler {
	return &VendorHandler{Pool: pool}
}

func (h *VendorHandler) GetVendors(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")

	if status != "" && status != "Active" && status != "Inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'Active' or 'Inactive'"})
		return
	}

	vendors, err := db.GetVendors(c.Request.Context(), h.Pool, db.VendorFilters{
		Status: status,
		Search: search,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vendors"})

		return
	}
	c.JSON(http.StatusOK, gin.H{"vendors": vendors})

}
