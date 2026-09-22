package handlers

import (
	"net/http"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type DashboardHandler struct {
	Pool *pgxpool.Pool
}

func NewDashboardHandler(pool *pgxpool.Pool) *DashboardHandler {
	return &DashboardHandler{Pool: pool}
}

func (h *DashboardHandler) GetOverview(c *gin.Context) {
	filters := db.DashboardFilters{
		VendorID: c.Query("vendorId"),
		From:     c.Query("from"),
		To:       c.Query("to"),
	}

	// overview, err := db.GetDashboardOverview(c.Request.Context(), h.Pool, filters)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch dashboard overview"})
	// 	return
	// }

	overview, err := db.GetDashboardOverview(c.Request.Context(), h.Pool, filters)
	if err != nil {
		log.Printf("Dashboard overview error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch dashboard overview",
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}
