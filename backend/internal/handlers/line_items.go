package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type LineItemHandler struct {
	Pool *pgxpool.Pool
}

func NewLineItemHandler(pool *pgxpool.Pool) *LineItemHandler {
	return &LineItemHandler{Pool: pool}
}

func (h *LineItemHandler) GetLineItems(c *gin.Context) {
	poNumber := c.Param("id")

	items, err := db.GetLineItems(c.Request.Context(), h.Pool, poNumber)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch line items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"line_items": items})
}

func (h *LineItemHandler) AddLineItem(c *gin.Context) {
	poNumber := c.Param("id")

	var req createSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var charges []db.CreateChargeInput
	for _, ch := range req.Charges {
		charges = append(charges, db.CreateChargeInput{
			ChargeTypeID: ch.ChargeTypeID,
			RatePerPiece: ch.RatePerPiece,
		})
	}

	result, err := db.AddLineItem(c.Request.Context(), h.Pool, poNumber, db.CreateSKUInput{
		SKUCode:             req.SKUCode,
		ProductName:         req.ProductName,
		Quantity:            req.Quantity,
		RatePerUnit:         req.RatePerUnit,
		PackagingFlat:       req.PackagingFlat,
		SellingPricePerUnit: req.SellingPricePerUnit,
		Charges:             charges,
	})
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add line item"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

type updateLineItemRequest struct {
	SKUCode             *string  `json:"sku_code"`
	ProductName         *string  `json:"product_name"`
	Quantity            *int     `json:"quantity"`
	RatePerUnit         *float64 `json:"rate_per_unit"`
	PackagingFlat       *float64 `json:"packaging_flat"`
	SellingPricePerUnit *float64 `json:"selling_price_per_unit"`
}

func (h *LineItemHandler) UpdateLineItem(c *gin.Context) {
	idParam := c.Param("id")
	lineItemID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line item id"})
		return
	}

	var req updateLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.UpdateLineItem(c.Request.Context(), h.Pool, lineItemID, db.UpdateLineItemInput{
		SKUCode:             req.SKUCode,
		ProductName:         req.ProductName,
		Quantity:            req.Quantity,
		RatePerUnit:         req.RatePerUnit,
		PackagingFlat:       req.PackagingFlat,
		SellingPricePerUnit: req.SellingPricePerUnit,
	})
	if err != nil {
		if err == db.ErrLineItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "line item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update line item"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *LineItemHandler) DeleteLineItem(c *gin.Context) {
	idParam := c.Param("id")
	lineItemID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line item id"})
		return
	}

	err = db.DeleteLineItem(c.Request.Context(), h.Pool, lineItemID)
	if err != nil {
		if err == db.ErrLineItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "line item not found"})
			return
		}
		if err == db.ErrCannotDeleteLastLineItem {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last remaining line item on a purchase order"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete line item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "line item deleted"})
}
