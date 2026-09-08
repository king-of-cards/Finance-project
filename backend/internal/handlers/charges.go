package handlers

import (
	"net/http"
	"strconv"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type ChargeHandler struct {
	Pool *pgxpool.Pool
}

func NewChargeHandler(pool *pgxpool.Pool) *ChargeHandler {
	return &ChargeHandler{Pool: pool}
}

// --- Line item charges ---

func (h *ChargeHandler) GetLineItemCharges(c *gin.Context) {
	lineItemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line item id"})
		return
	}

	charges, err := db.GetLineItemCharges(c.Request.Context(), h.Pool, lineItemID)
	if err != nil {
		if err == db.ErrLineItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "line item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch charges"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"charges": charges})
}

type replaceChargesRequest struct {
	Charges []createChargeRequest `json:"charges" binding:"required"`
}

func (h *ChargeHandler) ReplaceLineItemCharges(c *gin.Context) {
	lineItemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line item id"})
		return
	}

	var req replaceChargesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var charges []db.CreateChargeInput
	for _, ch := range req.Charges {
		charges = append(charges, db.CreateChargeInput{ChargeTypeID: ch.ChargeTypeID, RatePerPiece: ch.RatePerPiece})
	}

	result, err := db.ReplaceLineItemCharges(c.Request.Context(), h.Pool, lineItemID, charges)
	if err != nil {
		if err == db.ErrLineItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "line item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to replace charges"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"charges": result})
}

func (h *ChargeHandler) DeleteLineItemCharge(c *gin.Context) {
	lineItemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line item id"})
		return
	}
	chargeID, err := strconv.ParseInt(c.Param("chargeId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid charge id"})
		return
	}

	err = db.DeleteLineItemCharge(c.Request.Context(), h.Pool, lineItemID, chargeID)
	if err != nil {
		if err == db.ErrLineItemNotFound || err == db.ErrChargeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "charge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete charge"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "charge deleted"})
}

// --- Charge types ---

func (h *ChargeHandler) GetChargeTypes(c *gin.Context) {
	types, err := db.GetChargeTypes(c.Request.Context(), h.Pool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch charge types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"charge_types": types})
}

type createChargeTypeRequest struct {
	ChargeTypeID   string  `json:"charge_type_id" binding:"required"`
	Name           string  `json:"name" binding:"required"`
	DefaultRateMin float64 `json:"default_rate_min"`
	DefaultRateMax float64 `json:"default_rate_max"`
}

func (h *ChargeHandler) CreateChargeType(c *gin.Context) {
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	if !strings.EqualFold(role, "admin") && !strings.EqualFold(role, "approver") {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can create charge types"})
		return
	}

	var req createChargeTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ct, err := db.CreateChargeType(c.Request.Context(), h.Pool, req.ChargeTypeID, req.Name, req.DefaultRateMin, req.DefaultRateMax)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create charge type"})
		return
	}

	c.JSON(http.StatusCreated, ct)
}

type updateChargeTypeRequest struct {
	Name           *string  `json:"name"`
	DefaultRateMin *float64 `json:"default_rate_min"`
	DefaultRateMax *float64 `json:"default_rate_max"`
}

func (h *ChargeHandler) UpdateChargeType(c *gin.Context) {
	chargeTypeID := c.Param("id")

	var req updateChargeTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ct, err := db.UpdateChargeType(c.Request.Context(), h.Pool, chargeTypeID, db.UpdateChargeTypeInput{
		Name:           req.Name,
		DefaultRateMin: req.DefaultRateMin,
		DefaultRateMax: req.DefaultRateMax,
	})
	if err != nil {
		if err == db.ErrChargeTypeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "charge type not found"})
			return
		}
		if err == db.ErrNoFieldsToUpdate {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields provided to update"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update charge type"})
		return
	}

	c.JSON(http.StatusOK, ct)
}

func (h *ChargeHandler) DeleteChargeType(c *gin.Context) {
	chargeTypeID := c.Param("id")

	err := db.DeleteChargeType(c.Request.Context(), h.Pool, chargeTypeID)
	if err != nil {
		if err == db.ErrChargeTypeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "charge type not found"})
			return
		}
		if err == db.ErrChargeTypeInUse {
			c.JSON(http.StatusConflict, gin.H{"error": "charge type is in use and cannot be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete charge type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "charge type deleted"})
}
