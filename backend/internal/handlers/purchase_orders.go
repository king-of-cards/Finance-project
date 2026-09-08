package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/db"
)

type PurchaseOrderHandler struct {
	Pool *pgxpool.Pool
}

func NewPurchaseOrderHandler(pool *pgxpool.Pool) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{Pool: pool}
}

var validPOStatuses = map[string]bool{
	"PO Raised": true, "Waiting for Vendor": true, "Received": true, "Verified": true,
	"Pending Approval": true, "Approved": true, "Paid": true, "Rejected": true,
	"Cancelled": true, "Hold": true,
}

var validPaymentStatuses = map[string]bool{
	"Unpaid": true, "Payment Ready": true, "Paid": true,
}

func (h *PurchaseOrderHandler) GetPurchaseOrders(c *gin.Context) {
	status := c.Query("status")
	if status != "" && !validPOStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status value"})
		return
	}

	paymentStatus := c.Query("paymentStatus")
	if paymentStatus != "" && !validPaymentStatuses[paymentStatus] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paymentStatus value"})
		return
	}

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

	filters := db.PurchaseOrderFilters{
		Status:        status,
		PaymentStatus: paymentStatus,
		VendorID:      c.Query("vendorId"),
		From:          c.Query("from"),
		To:            c.Query("to"),
		Search:        c.Query("search"),
	}

	result, err := db.GetPurchaseOrders(c.Request.Context(), h.Pool, filters, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail   to fetch purchase orders"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PurchaseOrderHandler) ExportPurchaseOrders(c *gin.Context) {
	status := c.Query("status")
	if status != "" && !validPOStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status value"})
		return
	}

	paymentStatus := c.Query("paymentStatus")
	if paymentStatus != "" && !validPaymentStatuses[paymentStatus] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paymentStatus value"})
		return
	}

	filters := db.PurchaseOrderFilters{
		Status:        status,
		PaymentStatus: paymentStatus,
		VendorID:      c.Query("vendorId"),
		From:          c.Query("from"),
		To:            c.Query("to"),
		Search:        c.Query("search"),
	}

	filename := fmt.Sprintf("purchase_orders_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	if err := db.ExportPurchaseOrders(c.Request.Context(), h.Pool, filters, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export purchase orders"})
		return
	}
}

func (h *PurchaseOrderHandler) GetPurchaseOrderDetail(c *gin.Context) {
	poNumber := c.Param("id")

	detail, err := db.GetPurchaseOrderDetail(c.Request.Context(), h.Pool, poNumber)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch purchase order detail"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

type createChargeRequest struct {
	ChargeTypeID string  `json:"charge_type_id" binding:"required"`
	RatePerPiece float64 `json:"rate_per_piece"`
}

type createSKURequest struct {
	SKUCode             *string               `json:"sku_code"`
	ProductName         string                `json:"product_name" binding:"required"`
	Quantity            int                   `json:"quantity" binding:"required,gt=0"`
	RatePerUnit         float64               `json:"rate_per_unit" binding:"required,gte=0"`
	PackagingFlat       float64               `json:"packaging_flat"`
	SellingPricePerUnit float64               `json:"selling_price_per_unit"`
	Charges             []createChargeRequest `json:"charges"`
}

type createPurchaseOrderRequest struct {
	CustomerOrderNo      string             `json:"customer_order_no" binding:"required"`
	VendorID             string             `json:"vendor_id" binding:"required"`
	GSTPct               float64            `json:"gst_pct"`
	OrderedDate          *string            `json:"ordered_date"`
	ExpectedDeliveryDate *string            `json:"expected_delivery_date"`
	Remarks              *string            `json:"remarks"`
	SKUs                 []createSKURequest `json:"skus" binding:"required,min=1"`
}

func (h *PurchaseOrderHandler) CreatePurchaseOrder(c *gin.Context) {
	var req createPurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	createdBy := userIDVal.(string)

	if req.GSTPct == 0 {
		req.GSTPct = 18
	}

	var skus []db.CreateSKUInput
	for _, s := range req.SKUs {
		var charges []db.CreateChargeInput
		for _, ch := range s.Charges {
			charges = append(charges, db.CreateChargeInput{
				ChargeTypeID: ch.ChargeTypeID,
				RatePerPiece: ch.RatePerPiece,
			})
		}
		skus = append(skus, db.CreateSKUInput{
			SKUCode:             s.SKUCode,
			ProductName:         s.ProductName,
			Quantity:            s.Quantity,
			RatePerUnit:         s.RatePerUnit,
			PackagingFlat:       s.PackagingFlat,
			SellingPricePerUnit: s.SellingPricePerUnit,
			Charges:             charges,
		})
	}

	result, err := db.CreatePurchaseOrder(c.Request.Context(), h.Pool, db.CreatePurchaseOrderInput{
		CustomerOrderNo:      req.CustomerOrderNo,
		VendorID:             req.VendorID,
		CreatedBy:            createdBy,
		GSTPct:               req.GSTPct,
		OrderedDate:          req.OrderedDate,
		ExpectedDeliveryDate: req.ExpectedDeliveryDate,
		Remarks:              req.Remarks,
		SKUs:                 skus,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create purchase order"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

type updateChargeRequest struct {
	ChargeID     *int64   `json:"charge_id"`
	ChargeTypeID *string  `json:"charge_type_id"`
	RatePerPiece *float64 `json:"rate_per_piece"`
}

type updateSKURequest struct {
	LineItemID          *int64                `json:"line_item_id"`
	SKUCode             *string               `json:"sku_code"`
	ProductName         *string               `json:"product_name"`
	Quantity            *int                  `json:"quantity"`
	RatePerUnit         *float64              `json:"rate_per_unit"`
	PackagingFlat       *float64              `json:"packaging_flat"`
	SellingPricePerUnit *float64              `json:"selling_price_per_unit"`
	Charges             []updateChargeRequest `json:"charges"`
}

type updatePurchaseOrderRequest struct {
	PONumber             *string            `json:"po_number"`
	CustomerOrderNo      *string            `json:"customer_order_no"`
	VendorID             *string            `json:"vendor_id"`
	Status               *string            `json:"status"`
	PaymentStatus        *string            `json:"payment_status"`
	GSTPct               *float64           `json:"gst_pct"`
	OrderedDate          *string            `json:"ordered_date"`
	ExpectedDeliveryDate *string            `json:"expected_delivery_date"`
	Remarks              *string            `json:"remarks"`
	SKUs                 []updateSKURequest `json:"skus"`
}

func (h *PurchaseOrderHandler) UpdatePurchaseOrder(c *gin.Context) {
	poNumber := c.Param("id")

	var req updatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != nil && !validPOStatuses[*req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status value"})
		return
	}
	if req.PaymentStatus != nil && !validPaymentStatuses[*req.PaymentStatus] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paymentStatus value"})
		return
	}

	var skus []db.UpdateSKUInput
	for _, s := range req.SKUs {
		var charges []db.UpdateChargeInput
		for _, ch := range s.Charges {
			charges = append(charges, db.UpdateChargeInput{
				ChargeID:     ch.ChargeID,
				ChargeTypeID: ch.ChargeTypeID,
				RatePerPiece: ch.RatePerPiece,
			})
		}
		skus = append(skus, db.UpdateSKUInput{
			LineItemID:          s.LineItemID,
			SKUCode:             s.SKUCode,
			ProductName:         s.ProductName,
			Quantity:            s.Quantity,
			RatePerUnit:         s.RatePerUnit,
			PackagingFlat:       s.PackagingFlat,
			SellingPricePerUnit: s.SellingPricePerUnit,
			Charges:             charges,
		})
	}

	result, err := db.UpdatePurchaseOrder(c.Request.Context(), h.Pool, poNumber, db.UpdatePurchaseOrderInput{
		NewPONumber:          req.PONumber,
		CustomerOrderNo:      req.CustomerOrderNo,
		VendorID:             req.VendorID,
		Status:               req.Status,
		PaymentStatus:        req.PaymentStatus,
		GSTPct:               req.GSTPct,
		OrderedDate:          req.OrderedDate,
		ExpectedDeliveryDate: req.ExpectedDeliveryDate,
		Remarks:              req.Remarks,
		SKUs:                 skus,
	})
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		if err == db.ErrPONumberAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "po_number already in use"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update purchase order"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PurchaseOrderHandler) CancelPurchaseOrder(c *gin.Context) {
	poNumber := c.Param("id")

	result, err := db.CancelPurchaseOrder(c.Request.Context(), h.Pool, poNumber)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		if err == db.ErrPurchaseOrderLocked {
			c.JSON(http.StatusConflict, gin.H{"error": "purchase order is locked and cannot be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel purchase order"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// func (h *PurchaseOrderHandler) getPurchaseOrderStatusHistory(c *gin.Context) {
// 	poNumber := c.Param("id")

// 	history, err := db.GetPurchaseOrderStatusHistory(c.Request.Context(), h.Pool, poNumber)
// 	if err != nil {
// 		if err == db.ErrPurchaseOrderNotFound {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "failed to fetch status history"})
// 			return
// 		}
// 	}
// 	c.JSON(http.StatusOK, gin.H{"history": history})
// }

func (h *PurchaseOrderHandler) GetPurchaseOrderStatusHistory(c *gin.Context) {
	poNumber := c.Param("id")

	history, err := db.GetPurchaseOrderStatusHistory(c.Request.Context(), h.Pool, poNumber)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch status history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

func requireApproverRole(c *gin.Context) (string, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return "", false
	}
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	if role != "approver" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only approvers can perform this action"})
		return "", false
	}
	return userIDVal.(string), true
}

type approveRequest struct {
	VerifiedLineItemIds []int64 `json:"verifiedLineItemIds"`
	Comment             *string `json:"comment"`
}

func (h *PurchaseOrderHandler) ApprovePurchaseOrder(c *gin.Context) {
	userID, ok := requireApproverRole(c)
	if !ok {
		return
	}
	poNumber := c.Param("id")

	var req approveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := db.ApprovePurchaseOrder(c.Request.Context(), h.Pool, poNumber, req.VerifiedLineItemIds, req.Comment, userID)
	if err != nil {
		switch err {
		case db.ErrPurchaseOrderNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
		case db.ErrLineItemsNotFullyVerified:
			c.JSON(http.StatusBadRequest, gin.H{"error": "all line items must be verified before approval"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve purchase order"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

type reasonRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *PurchaseOrderHandler) RejectPurchaseOrder(c *gin.Context) {
	userID, ok := requireApproverRole(c)
	if !ok {
		return
	}
	poNumber := c.Param("id")

	var req reasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}

	result, err := db.RejectPurchaseOrder(c.Request.Context(), h.Pool, poNumber, req.Reason, userID)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject purchase order"})
		return
	}

	c.JSON(http.StatusOK, result)
}

type commentRequest struct {
	Comment *string `json:"comment"`
}

func (h *PurchaseOrderHandler) HoldPurchaseOrder(c *gin.Context) {
	userID, ok := requireApproverRole(c)
	if !ok {
		return
	}
	poNumber := c.Param("id")

	var req commentRequest
	_ = c.ShouldBindJSON(&req) // comment is optional, ignore bind errors on empty body

	result, err := db.HoldPurchaseOrder(c.Request.Context(), h.Pool, poNumber, userID, req.Comment)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hold purchase order"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PurchaseOrderHandler) RequestChangesPurchaseOrder(c *gin.Context) {
	userID, ok := requireApproverRole(c)
	if !ok {
		return
	}
	poNumber := c.Param("id")

	var req commentRequest
	_ = c.ShouldBindJSON(&req)

	result, err := db.RequestChangesPurchaseOrder(c.Request.Context(), h.Pool, poNumber, userID, req.Comment)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to request changes"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PurchaseOrderHandler) MarkPaidPurchaseOrder(c *gin.Context) {
	userID, ok := requireApproverRole(c)
	if !ok {
		return
	}
	poNumber := c.Param("id")

	var req commentRequest
	_ = c.ShouldBindJSON(&req)

	result, err := db.MarkPaidPurchaseOrder(c.Request.Context(), h.Pool, poNumber, userID, req.Comment)
	if err != nil {
		if err == db.ErrPurchaseOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark purchase order as paid"})
		return
	}

	c.JSON(http.StatusOK, result)
}
