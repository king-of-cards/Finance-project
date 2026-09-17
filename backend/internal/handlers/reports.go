package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jung-kurt/gofpdf"

	"github.com/king-of-cards/finance-project/internal/db"
)

type ReportsHandler struct {
	Pool *pgxpool.Pool
}

func NewReportsHandler(pool *pgxpool.Pool) *ReportsHandler {
	return &ReportsHandler{Pool: pool}
}

func (h *ReportsHandler) ExportReport(c *gin.Context) {
	reportType := c.Query("type")
	format := c.DefaultQuery("format", "csv")

	if reportType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type query param is required"})
		return
	}
	if format != "csv" && format != "pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be 'csv' or 'pdf'"})
		return
	}

	filters := db.PurchaseOrderFilters{
		VendorID: c.Query("vendorId"),
		From:     c.Query("from"),
		To:       c.Query("to"),
	}

	report, err := db.GetReport(c.Request.Context(), h.Pool, reportType, filters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid report type: %s", reportType)})
		return
	}

	timestamp := time.Now().Format("20060102_150405")

	if format == "csv" {
		filename := fmt.Sprintf("report_%s_%s.csv", reportType, timestamp)
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

		csvWriter := csv.NewWriter(c.Writer)
		csvWriter.Write(report.Headers)
		for _, row := range report.Rows {
			csvWriter.Write([]string(row))
		}
		csvWriter.Flush()
		return
	}

	// PDF
	filename := fmt.Sprintf("report_%s_%s.pdf", reportType, timestamp)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, fmt.Sprintf("Report: %s", reportType), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 9)

	colWidth := 277.0 / float64(len(report.Headers))
	for _, header := range report.Headers {
		pdf.CellFormat(colWidth, 7, header, "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 8)
	for _, row := range report.Rows {
		for _, cell := range row {
			pdf.CellFormat(colWidth, 6, cell, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	if err := pdf.Output(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate PDF"})
		return
	}
}
