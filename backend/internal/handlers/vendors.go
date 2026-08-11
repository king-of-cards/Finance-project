package handlers

import (
	"errors"
	"net/http"
	"strconv"

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

func (h *VendorHandler) GetVendorDetail(c *gin.Context) {
	vendorID := c.Param("id")

	detail, err := db.GetVendorDetail(c.Request.Context(), h.Pool, vendorID)

	if err != nil {
		if err == db.ErrVendorNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "vendor not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vendor detail"})
		return
	}
	c.JSON(http.StatusOK, detail)

}

func (h *VendorHandler) GetVendorOrders(c *gin.Context) {
	vendorID := c.Param("id")

	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed >= 1 {
			page = parsed
		}
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed >= 1 && parsed <= 100 {
			limit = parsed
		}
	}

	result, err := db.GetVendorOrders(c.Request.Context(), h.Pool, vendorID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vendor orders"})
		return
	}

	c.JSON(http.StatusOK, result)
}

var validPaymentTerms = map[string]bool{
	"Net 15":      true,
	"Net 30":      true,
	"Net 45":      true,
	"Advance 50%": true,
	"On Delivery": true,
}

type createVendorRequest struct {
	Name          string  `json:"name" binding:"required"`
	City          *string `json:"city"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Address       *string `json:"address"`
	GSTNumber     *string `json:"gst_number"`
	PANNumber     *string `json:"pan_number"`
	PaymentTerms  string  `json:"payment_terms"`
}

func (h *VendorHandler) CreateVendor(c *gin.Context) {
	var req createVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "name is required",
		})
		return
	}

	if req.PaymentTerms != "" && !validPaymentTerms[req.PaymentTerms] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment_terms value"})
		return
	}

	vendor, err := db.CreateVendor(c.Request.Context(), h.Pool, db.CreateVendorInput{
		Name:          req.Name,
		City:          req.City,
		ContactPerson: req.ContactPerson,
		Phone:         req.Phone,
		Email:         req.Email,
		Address:       req.Address,
		GSTNumber:     req.GSTNumber,
		PANNumber:     req.PANNumber,
		PaymentTerms:  req.PaymentTerms,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create vendor"})
		return
	}
	c.JSON(http.StatusCreated, vendor)

}

type updateVendorRequest struct {
	Name          *string `json:"name"`
	City          *string `json:"city"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Address       *string `json:"address"`
	GSTNumber     *string `json:"gst_number"`
	PANNumber     *string `json:"pan_number"`
	PaymentTerms  *string `json:"payment_terms"`
	Status        *string `json:"status"`
	Rating        *int16  `json:"rating"`
}

var ErrNoFieldsToUpdate = errors.New("no fields provided to update")

func (h *VendorHandler) UpdateVendor(c *gin.Context) {
	vendorID := c.Param("id")

	var req updateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.PaymentTerms != nil && !validPaymentTerms[*req.PaymentTerms] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment_terms value"})
		return
	}

	if req.Status != nil && *req.Status != "Active" && *req.Status != "Inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'Active' or 'Inactive'"})
		return
	}

	if req.Rating != nil && (*req.Rating < 1 || *req.Rating > 5) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
		return
	}

	vendor, err := db.UpdateVendor(c.Request.Context(), h.Pool, vendorID, db.UpdateVendorInput{
		Name:          req.Name,
		City:          req.City,
		ContactPerson: req.ContactPerson,
		Phone:         req.Phone,
		Email:         req.Email,
		Address:       req.Address,
		GSTNumber:     req.GSTNumber,
		PANNumber:     req.PANNumber,
		PaymentTerms:  req.PaymentTerms,
		Status:        req.Status,
		Rating:        req.Rating,
	})

	if err != nil {
		if err == db.ErrVendorNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "vendor not found"})
			return
		}
		if err == db.ErrNoFieldsToUpdate {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields provided to update"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vendor"})
		return
	}

	c.JSON(http.StatusOK, vendor)

}
