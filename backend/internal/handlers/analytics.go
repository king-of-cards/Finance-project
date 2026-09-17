package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type AnalyticsHandler struct {
	Pool *pgxpool.Pool
}

func NewAnalyticsHandler(pool *pgxpool.Pool) *AnalyticsHandler {
	return &AnalyticsHandler{Pool: pool}
}

func (h *AnalyticsHandler) GetOverview(c *gin.Context) {
	filters := db.DashboardFilters{
		VendorID: c.Query("vendorId"),
		From:     c.Query("from"),
		To:       c.Query("to"),
	}

	overview, err := db.GetAnalyticsOverview(c.Request.Context(), h.Pool, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analytics overview"})
		return
	}

	c.JSON(http.StatusOK, overview)
}
