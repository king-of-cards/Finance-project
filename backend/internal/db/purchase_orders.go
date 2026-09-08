package db

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseOrder struct {
	PONumber             string     `json:"po_number"`
	CustomerOrderNo      string     `json:"customer_order_no"`
	VendorID             string     `json:"vendor_id"`
	VendorName           string     `json:"vendor_name"`
	Status               string     `json:"status"`
	PaymentStatus        string     `json:"payment_status"`
	OrderedDate          *time.Time `json:"ordered_date"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`
	InvoiceNo            *string    `json:"invoice_no"`
	LandingCost          float64    `json:"landing_cost"`
	GrossMarginPct       float64    `json:"gross_margin_pct"`
	TotalQty             int        `json:"total_qty"`
}

type PurchaseOrderFilters struct {
	Status        string
	PaymentStatus string
	VendorID      string
	From          string
	To            string
	Search        string
}

type PaginatedPurchaseOrders struct {
	Orders     []PurchaseOrder `json:"orders"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	Total      int             `json:"total"`
	TotalPages int             `json:"total_pages"`
}

func buildPurchaseOrderQuery(filters PurchaseOrderFilters, selectClause string) (string, []interface{}) {
	query := selectClause + `
		FROM finance_purchase_orders po
		JOIN finance_vendors v ON v.vendor_id = po.vendor_id
	`

	var conditions []string
	var args []interface{}
	argPos := 1

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("po.status = $%d", argPos))
		args = append(args, filters.Status)
		argPos++
	}

	if filters.PaymentStatus != "" {
		conditions = append(conditions, fmt.Sprintf("po.payment_status = $%d", argPos))
		args = append(args, filters.PaymentStatus)
		argPos++
	}

	if filters.VendorID != "" {
		conditions = append(conditions, fmt.Sprintf("po.vendor_id = $%d", argPos))
		args = append(args, filters.VendorID)
		argPos++
	}

	if filters.From != "" {
		conditions = append(conditions, fmt.Sprintf("po.ordered_date >= $%d", argPos))
		args = append(args, filters.From)
		argPos++
	}

	if filters.To != "" {
		conditions = append(conditions, fmt.Sprintf("po.ordered_date <= $%d", argPos))
		args = append(args, filters.To)
		argPos++
	}

	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		searchCondition := fmt.Sprintf(`(
			po.po_number ILIKE $%d OR
			po.customer_order_no ILIKE $%d OR
			po.invoice_no ILIKE $%d OR
			v.name ILIKE $%d OR
			EXISTS (
				SELECT 1 FROM finance_order_line_items li
				WHERE li.po_number = po.po_number
				AND (li.sku_code ILIKE $%d OR li.product_name ILIKE $%d)
			)
		)`, argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5)
		conditions = append(conditions, searchCondition)
		for i := 0; i < 6; i++ {
			args = append(args, searchTerm)
		}
		argPos += 6
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}

func GetPurchaseOrders(ctx context.Context, pool *pgxpool.Pool, filters PurchaseOrderFilters, page, limit int) (*PaginatedPurchaseOrders, error) {
	countQuery, countArgs := buildPurchaseOrderQuery(filters, "SELECT COUNT(*)")

	var total int
	err := pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("counting purchase orders: %w", err)
	}

	selectClause := `SELECT po.po_number, po.customer_order_no, po.vendor_id, v.name,
		po.status, po.payment_status, po.ordered_date, po.expected_delivery_date,
		po.invoice_no, po.landing_cost, po.gross_margin_pct, po.total_qty`

	query, args := buildPurchaseOrderQuery(filters, selectClause)

	offset := (page - 1) * limit
	argPos := len(args) + 1
	query += fmt.Sprintf(" ORDER BY po.ordered_date DESC NULLS LAST LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying purchase orders: %w", err)
	}
	defer rows.Close()

	var orders []PurchaseOrder
	for rows.Next() {
		var o PurchaseOrder
		if err := rows.Scan(
			&o.PONumber, &o.CustomerOrderNo, &o.VendorID, &o.VendorName,
			&o.Status, &o.PaymentStatus, &o.OrderedDate, &o.ExpectedDeliveryDate,
			&o.InvoiceNo, &o.LandingCost, &o.GrossMarginPct, &o.TotalQty,
		); err != nil {
			return nil, fmt.Errorf("scanning purchase order row: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating purchase order rows: %w", err)
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedPurchaseOrders{
		Orders:     orders,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

type ChargeDetail struct {
	ChargeID     int64   `json:"charge_id"`
	ChargeTypeID string  `json:"charge_type_id"`
	ChargeName   string  `json:"charge_name"`
	RatePerPiece float64 `json:"rate_per_piece"`
}

type LineItemDetail struct {
	LineItemID          int64          `json:"line_item_id"`
	SKUCode             *string        `json:"sku_code"`
	ProductName         string         `json:"product_name"`
	Quantity            int            `json:"quantity"`
	RatePerUnit         float64        `json:"rate_per_unit"`
	PackagingFlat       float64        `json:"packaging_flat"`
	SellingPricePerUnit float64        `json:"selling_price_per_unit"`
	Charges             []ChargeDetail `json:"charges"`
	ComputedLineTotal   float64        `json:"computed_line_total"`
}

type PurchaseOrderDetail struct {
	PurchaseOrder
	LineItems          []LineItemDetail `json:"line_items"`
	ComputedGrandTotal float64          `json:"computed_grand_total"`
	StoredLandingCost  float64          `json:"stored_landing_cost"`
	Discrepancy        float64          `json:"discrepancy"`
}

var ErrPurchaseOrderNotFound = errors.New("purchase order not found")

func GetPurchaseOrderDetail(ctx context.Context, pool *pgxpool.Pool, poNumber string) (*PurchaseOrderDetail, error) {
	var d PurchaseOrderDetail

	err := pool.QueryRow(ctx, `
		SELECT po.po_number, po.customer_order_no, po.vendor_id, v.name,
		       po.status, po.payment_status, po.ordered_date, po.expected_delivery_date,
		       po.invoice_no, po.landing_cost, po.gross_margin_pct, po.total_qty
		FROM finance_purchase_orders po
		JOIN finance_vendors v ON v.vendor_id = po.vendor_id
		WHERE po.po_number = $1
	`, poNumber).Scan(
		&d.PONumber, &d.CustomerOrderNo, &d.VendorID, &d.VendorName,
		&d.Status, &d.PaymentStatus, &d.OrderedDate, &d.ExpectedDeliveryDate,
		&d.InvoiceNo, &d.LandingCost, &d.GrossMarginPct, &d.TotalQty,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying purchase order header: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT line_item_id, sku_code, product_name, quantity, rate_per_unit,
		       packaging_flat, selling_price_per_unit
		FROM finance_order_line_items
		WHERE po_number = $1
		ORDER BY line_item_id
	`, poNumber)
	if err != nil {
		return nil, fmt.Errorf("querying line items: %w", err)
	}
	defer rows.Close()

	var lineItemIDs []int64
	itemsByID := map[int64]*LineItemDetail{}

	for rows.Next() {
		var li LineItemDetail
		if err := rows.Scan(
			&li.LineItemID, &li.SKUCode, &li.ProductName, &li.Quantity,
			&li.RatePerUnit, &li.PackagingFlat, &li.SellingPricePerUnit,
		); err != nil {
			return nil, fmt.Errorf("scanning line item row: %w", err)
		}
		li.Charges = []ChargeDetail{}
		d.LineItems = append(d.LineItems, li)
		lineItemIDs = append(lineItemIDs, li.LineItemID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating line item rows: %w", err)
	}

	for i := range d.LineItems {
		itemsByID[d.LineItems[i].LineItemID] = &d.LineItems[i]
	}

	if len(lineItemIDs) > 0 {
		chargeRows, err := pool.Query(ctx, `
			SELECT c.line_item_id, c.charge_id, c.charge_type_id, ct.name, c.rate_per_piece
			FROM finance_line_item_charges c
			JOIN finance_charge_types ct ON ct.charge_type_id = c.charge_type_id
			WHERE c.line_item_id = ANY($1)
		`, lineItemIDs)
		if err != nil {
			return nil, fmt.Errorf("querying charges: %w", err)
		}
		defer chargeRows.Close()

		for chargeRows.Next() {
			var lineItemID int64
			var ch ChargeDetail
			if err := chargeRows.Scan(&lineItemID, &ch.ChargeID, &ch.ChargeTypeID, &ch.ChargeName, &ch.RatePerPiece); err != nil {
				return nil, fmt.Errorf("scanning charge row: %w", err)
			}
			if li, ok := itemsByID[lineItemID]; ok {
				li.Charges = append(li.Charges, ch)
			}
		}
		if err := chargeRows.Err(); err != nil {
			return nil, fmt.Errorf("iterating charge rows: %w", err)
		}
	}

	var grandTotal float64
	for i := range d.LineItems {
		li := &d.LineItems[i]
		qty := float64(li.Quantity)

		var chargesPerPiece float64
		for _, ch := range li.Charges {
			chargesPerPiece += ch.RatePerPiece
		}

		li.ComputedLineTotal = qty*li.RatePerUnit + qty*li.PackagingFlat + qty*chargesPerPiece
		grandTotal += li.ComputedLineTotal
	}

	d.ComputedGrandTotal = grandTotal
	d.StoredLandingCost = d.LandingCost
	d.Discrepancy = grandTotal - d.LandingCost

	return &d, nil
}

type CreateChargeInput struct {
	ChargeTypeID string  `json:"charge_type_id"`
	RatePerPiece float64 `json:"rate_per_piece"`
}

type CreateSKUInput struct {
	SKUCode             *string             `json:"sku_code"`
	ProductName         string              `json:"product_name"`
	Quantity            int                 `json:"quantity"`
	RatePerUnit         float64             `json:"rate_per_unit"`
	PackagingFlat       float64             `json:"packaging_flat"`
	SellingPricePerUnit float64             `json:"selling_price_per_unit"`
	Charges             []CreateChargeInput `json:"charges"`
}

type CreatePurchaseOrderInput struct {
	CustomerOrderNo      string
	VendorID             string
	CreatedBy            string
	GSTPct               float64
	OrderedDate          *string
	ExpectedDeliveryDate *string
	Remarks              *string
	SKUs                 []CreateSKUInput
}

func generateNextPONumber(ctx context.Context, tx pgx.Tx) (string, error) {
	var maxNum int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(CAST(SUBSTRING(po_number FROM 4) AS INTEGER)), 1000)
		FROM finance_purchase_orders
		WHERE po_number ~ '^PO-[0-9]+$'
	`).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("generating next po_number: %w", err)
	}
	return fmt.Sprintf("PO-%04d", maxNum+1), nil
}

func CreatePurchaseOrder(ctx context.Context, pool *pgxpool.Pool, input CreatePurchaseOrderInput) (*PurchaseOrderDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if already committed

	poNumber, err := generateNextPONumber(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Compute totals from the submitted SKUs and charges
	var baseTotal, packagingTotal, sellingTotal, otherChargesTotal float64
	var totalQty int

	type computedSKU struct {
		input      CreateSKUInput
		lineTotal  float64
		lineItemID int64
	}
	var computed []computedSKU

	for _, sku := range input.SKUs {
		qty := float64(sku.Quantity)
		lineBase := qty * sku.RatePerUnit
		linePackaging := qty * sku.PackagingFlat

		var lineChargesTotal float64
		for _, ch := range sku.Charges {
			lineChargesTotal += qty * ch.RatePerPiece
		}

		lineTotal := lineBase + linePackaging + lineChargesTotal

		baseTotal += lineBase
		packagingTotal += linePackaging
		otherChargesTotal += lineChargesTotal
		sellingTotal += qty * sku.SellingPricePerUnit
		totalQty += sku.Quantity

		computed = append(computed, computedSKU{input: sku, lineTotal: lineTotal})
	}

	gstAmount := baseTotal * (input.GSTPct / 100)
	grossAmount := baseTotal + packagingTotal + otherChargesTotal
	landingCost := grossAmount + gstAmount
	grossProfit := sellingTotal - landingCost

	var grossMarginPct float64
	if sellingTotal > 0 {
		grossMarginPct = (grossProfit / sellingTotal) * 100
	}

	var landingCostPerUnit float64
	if totalQty > 0 {
		landingCostPerUnit = landingCost / float64(totalQty)
	}

	var landingPct float64
	if sellingTotal > 0 {
		landingPct = (landingCost / sellingTotal) * 100
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO finance_purchase_orders
			(po_number, customer_order_no, vendor_id, created_by, status, payment_status,
			 gst_pct, ordered_date, expected_delivery_date, remarks,
			 base_total, packaging_total, other_charges_total, gross_amount, gst_amount,
			 landing_cost, total_qty, landing_cost_per_unit, selling_total, gross_profit,
			 gross_margin_pct, landing_pct)
		VALUES ($1, $2, $3, $4, 'PO Raised', 'Unpaid', $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, poNumber, input.CustomerOrderNo, input.VendorID, input.CreatedBy,
		input.GSTPct, input.OrderedDate, input.ExpectedDeliveryDate, input.Remarks,
		baseTotal, packagingTotal, otherChargesTotal, grossAmount, gstAmount,
		landingCost, totalQty, landingCostPerUnit, sellingTotal, grossProfit,
		grossMarginPct, landingPct)
	if err != nil {
		return nil, fmt.Errorf("inserting purchase order: %w", err)
	}

	for i, c := range computed {
		var lineItemID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO finance_order_line_items
				(po_number, sku_code, product_name, quantity, rate_per_unit, packaging_flat, selling_price_per_unit)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING line_item_id
		`, poNumber, c.input.SKUCode, c.input.ProductName, c.input.Quantity,
			c.input.RatePerUnit, c.input.PackagingFlat, c.input.SellingPricePerUnit,
		).Scan(&lineItemID)
		if err != nil {
			return nil, fmt.Errorf("inserting line item %d: %w", i, err)
		}
		computed[i].lineItemID = lineItemID

		for _, ch := range c.input.Charges {
			_, err = tx.Exec(ctx, `
				INSERT INTO finance_line_item_charges (line_item_id, charge_type_id, rate_per_piece)
				VALUES ($1, $2, $3)
			`, lineItemID, ch.ChargeTypeID, ch.RatePerPiece)
			if err != nil {
				return nil, fmt.Errorf("inserting charge for line item %d: %w", lineItemID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return GetPurchaseOrderDetail(ctx, pool, poNumber)
}

type UpdateChargeInput struct {
	ChargeID     *int64
	ChargeTypeID *string
	RatePerPiece *float64
}

type UpdateSKUInput struct {
	LineItemID          *int64
	SKUCode             *string
	ProductName         *string
	Quantity            *int
	RatePerUnit         *float64
	PackagingFlat       *float64
	SellingPricePerUnit *float64
	Charges             []UpdateChargeInput
}

type UpdatePurchaseOrderInput struct {
	NewPONumber          *string
	CustomerOrderNo      *string
	VendorID             *string
	Status               *string
	PaymentStatus        *string
	GSTPct               *float64
	OrderedDate          *string
	ExpectedDeliveryDate *string
	Remarks              *string
	SKUs                 []UpdateSKUInput
}

func UpdatePurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber string, input UpdatePurchaseOrderInput) (*PurchaseOrderDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	currentPONumber := poNumber

	// Step 1: handle po_number rename first (relies on ON UPDATE CASCADE)
	if input.NewPONumber != nil && *input.NewPONumber != poNumber {
		var exists bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_purchase_orders WHERE po_number = $1)`, *input.NewPONumber).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("checking new po_number: %w", err)
		}
		if exists {
			return nil, ErrPONumberAlreadyExists
		}

		_, err = tx.Exec(ctx, `UPDATE finance_purchase_orders SET po_number = $1 WHERE po_number = $2`, *input.NewPONumber, poNumber)
		if err != nil {
			return nil, fmt.Errorf("renaming po_number: %w", err)
		}
		currentPONumber = *input.NewPONumber
	}

	// Step 2: update header fields (dynamic SET, same pattern as UpdateVendor)
	var setClauses []string
	var args []interface{}
	argPos := 1
	addField := func(column string, value interface{}) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, argPos))
		args = append(args, value)
		argPos++
	}

	if input.CustomerOrderNo != nil {
		addField("customer_order_no", *input.CustomerOrderNo)
	}
	if input.VendorID != nil {
		addField("vendor_id", *input.VendorID)
	}
	if input.Status != nil {
		addField("status", *input.Status)
	}
	if input.PaymentStatus != nil {
		addField("payment_status", *input.PaymentStatus)
	}
	if input.GSTPct != nil {
		addField("gst_pct", *input.GSTPct)
	}
	if input.OrderedDate != nil {
		addField("ordered_date", *input.OrderedDate)
	}
	if input.ExpectedDeliveryDate != nil {
		addField("expected_delivery_date", *input.ExpectedDeliveryDate)
	}
	if input.Remarks != nil {
		addField("remarks", *input.Remarks)
	}

	if len(setClauses) > 0 {
		query := fmt.Sprintf(`UPDATE finance_purchase_orders SET %s WHERE po_number = $%d`,
			strings.Join(setClauses, ", "), argPos)
		args = append(args, currentPONumber)
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("updating purchase order header: %w", err)
		}
	}

	// Step 3: apply sku (line item) changes
	for _, sku := range input.SKUs {
		if sku.LineItemID != nil {
			// Update existing line item
			var liSet []string
			var liArgs []interface{}
			liPos := 1
			addLI := func(column string, value interface{}) {
				liSet = append(liSet, fmt.Sprintf("%s = $%d", column, liPos))
				liArgs = append(liArgs, value)
				liPos++
			}
			if sku.SKUCode != nil {
				addLI("sku_code", *sku.SKUCode)
			}
			if sku.ProductName != nil {
				addLI("product_name", *sku.ProductName)
			}
			if sku.Quantity != nil {
				addLI("quantity", *sku.Quantity)
			}
			if sku.RatePerUnit != nil {
				addLI("rate_per_unit", *sku.RatePerUnit)
			}
			if sku.PackagingFlat != nil {
				addLI("packaging_flat", *sku.PackagingFlat)
			}
			if sku.SellingPricePerUnit != nil {
				addLI("selling_price_per_unit", *sku.SellingPricePerUnit)
			}
			if len(liSet) > 0 {
				q := fmt.Sprintf(`UPDATE finance_order_line_items SET %s WHERE line_item_id = $%d`,
					strings.Join(liSet, ", "), liPos)
				liArgs = append(liArgs, *sku.LineItemID)
				if _, err := tx.Exec(ctx, q, liArgs...); err != nil {
					return nil, fmt.Errorf("updating line item %d: %w", *sku.LineItemID, err)
				}
			}

			for _, ch := range sku.Charges {
				if ch.ChargeID != nil {
					var chSet []string
					var chArgs []interface{}
					chPos := 1
					if ch.ChargeTypeID != nil {
						chSet = append(chSet, fmt.Sprintf("charge_type_id = $%d", chPos))
						chArgs = append(chArgs, *ch.ChargeTypeID)
						chPos++
					}
					if ch.RatePerPiece != nil {
						chSet = append(chSet, fmt.Sprintf("rate_per_piece = $%d", chPos))
						chArgs = append(chArgs, *ch.RatePerPiece)
						chPos++
					}
					if len(chSet) > 0 {
						q := fmt.Sprintf(`UPDATE finance_line_item_charges SET %s WHERE charge_id = $%d`,
							strings.Join(chSet, ", "), chPos)
						chArgs = append(chArgs, *ch.ChargeID)
						if _, err := tx.Exec(ctx, q, chArgs...); err != nil {
							return nil, fmt.Errorf("updating charge %d: %w", *ch.ChargeID, err)
						}
					}
				} else if ch.ChargeTypeID != nil && ch.RatePerPiece != nil {
					_, err := tx.Exec(ctx, `
						INSERT INTO finance_line_item_charges (line_item_id, charge_type_id, rate_per_piece)
						VALUES ($1, $2, $3)
					`, *sku.LineItemID, *ch.ChargeTypeID, *ch.RatePerPiece)
					if err != nil {
						return nil, fmt.Errorf("inserting new charge on line item %d: %w", *sku.LineItemID, err)
					}
				}
			}
		} else {
			// New line item
			if sku.ProductName == nil || sku.Quantity == nil || sku.RatePerUnit == nil {
				return nil, fmt.Errorf("new sku requires product_name, quantity, and rate_per_unit")
			}
			var newLineItemID int64
			packaging := 0.0
			if sku.PackagingFlat != nil {
				packaging = *sku.PackagingFlat
			}
			selling := 0.0
			if sku.SellingPricePerUnit != nil {
				selling = *sku.SellingPricePerUnit
			}
			err := tx.QueryRow(ctx, `
				INSERT INTO finance_order_line_items (po_number, sku_code, product_name, quantity, rate_per_unit, packaging_flat, selling_price_per_unit)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING line_item_id
			`, currentPONumber, sku.SKUCode, *sku.ProductName, *sku.Quantity, *sku.RatePerUnit, packaging, selling).Scan(&newLineItemID)
			if err != nil {
				return nil, fmt.Errorf("inserting new line item: %w", err)
			}

			for _, ch := range sku.Charges {
				if ch.ChargeTypeID != nil && ch.RatePerPiece != nil {
					_, err := tx.Exec(ctx, `
						INSERT INTO finance_line_item_charges (line_item_id, charge_type_id, rate_per_piece)
						VALUES ($1, $2, $3)
					`, newLineItemID, *ch.ChargeTypeID, *ch.RatePerPiece)
					if err != nil {
						return nil, fmt.Errorf("inserting charge on new line item: %w", err)
					}
				}
			}
		}
	}

	// Step 4: recompute totals from ALL current line items + charges for this PO
	if err := recomputePOTotals(ctx, tx, currentPONumber); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return GetPurchaseOrderDetail(ctx, pool, currentPONumber)
}

var ErrPONumberAlreadyExists = errors.New("po_number already exists")

func recomputePOTotals(ctx context.Context, tx pgx.Tx, poNumber string) error {
	rows, err := tx.Query(ctx, `
		SELECT li.line_item_id, li.quantity, li.rate_per_unit, li.packaging_flat, li.selling_price_per_unit,
		       COALESCE(SUM(c.rate_per_piece), 0) AS charges_per_piece
		FROM finance_order_line_items li
		LEFT JOIN finance_line_item_charges c ON c.line_item_id = li.line_item_id
		WHERE li.po_number = $1
		GROUP BY li.line_item_id, li.quantity, li.rate_per_unit, li.packaging_flat, li.selling_price_per_unit
	`, poNumber)
	if err != nil {
		return fmt.Errorf("recomputing totals - querying line items: %w", err)
	}
	defer rows.Close()

	var baseTotal, packagingTotal, otherChargesTotal, sellingTotal float64
	var totalQty int

	for rows.Next() {
		var qty int
		var rate, packaging, selling, chargesPerPiece float64
		var lineItemID int64
		if err := rows.Scan(&lineItemID, &qty, &rate, &packaging, &selling, &chargesPerPiece); err != nil {
			return fmt.Errorf("scanning line item for recompute: %w", err)
		}
		q := float64(qty)
		baseTotal += q * rate
		packagingTotal += q * packaging
		otherChargesTotal += q * chargesPerPiece
		sellingTotal += q * selling
		totalQty += qty
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating line items for recompute: %w", err)
	}

	var gstPct float64
	if err := tx.QueryRow(ctx, `SELECT gst_pct FROM finance_purchase_orders WHERE po_number = $1`, poNumber).Scan(&gstPct); err != nil {
		return fmt.Errorf("reading gst_pct for recompute: %w", err)
	}

	gstAmount := baseTotal * (gstPct / 100)
	grossAmount := baseTotal + packagingTotal + otherChargesTotal
	landingCost := grossAmount + gstAmount
	grossProfit := sellingTotal - landingCost

	var grossMarginPct, landingCostPerUnit, landingPct float64
	if sellingTotal > 0 {
		grossMarginPct = (grossProfit / sellingTotal) * 100
		landingPct = (landingCost / sellingTotal) * 100
	}
	if totalQty > 0 {
		landingCostPerUnit = landingCost / float64(totalQty)
	}

	_, err = tx.Exec(ctx, `
		UPDATE finance_purchase_orders
		SET base_total=$1, packaging_total=$2, other_charges_total=$3, gross_amount=$4, gst_amount=$5,
		    landing_cost=$6, total_qty=$7, landing_cost_per_unit=$8, selling_total=$9, gross_profit=$10,
		    gross_margin_pct=$11, landing_pct=$12
		WHERE po_number = $13
	`, baseTotal, packagingTotal, otherChargesTotal, grossAmount, gstAmount,
		landingCost, totalQty, landingCostPerUnit, sellingTotal, grossProfit,
		grossMarginPct, landingPct, poNumber)
	if err != nil {
		return fmt.Errorf("updating recomputed totals: %w", err)
	}

	return nil
}

var ErrPurchaseOrderLocked = errors.New("purchase order is locked and cannot be deleted")

func CancelPurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber string) (*PurchaseOrderDetail, error) {
	var locked bool
	err := pool.QueryRow(ctx, `
		SELECT locked FROM finance_purchase_orders WHERE po_number = $1
	`, poNumber).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("checking locked status: %w", err)
	}
	if locked {
		return nil, ErrPurchaseOrderLocked
	}

	_, err = pool.Exec(ctx, `
		UPDATE finance_purchase_orders SET status = 'Cancelled' WHERE po_number = $1
	`, poNumber)
	if err != nil {
		return nil, fmt.Errorf("cancelling purchase order: %w", err)
	}

	return GetPurchaseOrderDetail(ctx, pool, poNumber)

}

func ExportPurchaseOrders(ctx context.Context, pool *pgxpool.Pool, filters PurchaseOrderFilters, w io.Writer) error {
	selectClause := `SELECT po.po_number, po.customer_order_no, po.vendor_id, v.name,
		po.status, po.payment_status, po.ordered_date, po.expected_delivery_date,
		po.invoice_no, po.landing_cost, po.gross_margin_pct, po.total_qty`

	query, args := buildPurchaseOrderQuery(filters, selectClause)
	query += " ORDER BY po.ordered_date DESC NULLS LAST"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("querying purchase orders for export: %w", err)
	}

	defer rows.Close()

	csvWriter := csv.NewWriter(w)

	defer csvWriter.Flush()

	header := []string{
		"PO Number", "Customer Order No", "Vendor ID", "Vendor Name",
		"Status", "Payment Status", "Ordered Date", "Expected Delivery Date",
		"Invoice No", "Landing Cost", "Gross Margin %", "Total Qty",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for rows.Next() {
		var o PurchaseOrder
		if err := rows.Scan(
			&o.PONumber, &o.CustomerOrderNo, &o.VendorID, &o.VendorName,
			&o.Status, &o.PaymentStatus, &o.OrderedDate, &o.ExpectedDeliveryDate,
			&o.InvoiceNo, &o.LandingCost, &o.GrossMarginPct, &o.TotalQty,
		); err != nil {
			return fmt.Errorf("scanning purchase order row for export: %w", err)
		}

		record := []string{
			o.PONumber,
			o.CustomerOrderNo,
			o.VendorID,
			o.VendorName,
			o.Status,
			o.PaymentStatus,
			formatDatePtr(o.OrderedDate),
			formatDatePtr(o.ExpectedDeliveryDate),
			stringPtrOrEmpty(o.InvoiceNo),
			fmt.Sprintf("%.2f", o.LandingCost),
			fmt.Sprintf("%.2f", o.GrossMarginPct),
			fmt.Sprintf("%d", o.TotalQty),
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating purchase order rows for export: %w", err)
	}

	return nil

}

func formatDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s

}

type StatusHistoryEntry struct {
	HistoryID     int64     `json:"history_id"`
	FromStatus    *string   `json:"from_status"`
	ToStatus      string    `json:"to_status"`
	ChangedBy     string    `json:"changed_by"`
	ChangedByName string    `json:"changed_by_name"`
	Note          *string   `json:"note"`
	ChangedAt     time.Time `json:"changed_at"`
}

func GetPurchaseOrderStatusHistory(ctx context.Context, pool *pgxpool.Pool, poNumber string) ([]StatusHistoryEntry, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_purchase_orders WHERE po_number = $1)`, poNumber).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking purchase order existence: %w", err)
	}
	if !exists {
		return nil, ErrPurchaseOrderNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT h.history_id, h.from_status, h.to_status, h.changed_by, u.name, h.note, h.changed_at
		FROM finance_order_status_history h
		JOIN finance_users u ON u.user_id = h.changed_by
		WHERE h.po_number = $1
		ORDER BY h.changed_at ASC
	`, poNumber)
	if err != nil {
		return nil, fmt.Errorf("querying status history: %w", err)
	}
	defer rows.Close()

	var history []StatusHistoryEntry
	for rows.Next() {
		var e StatusHistoryEntry
		if err := rows.Scan(&e.HistoryID, &e.FromStatus, &e.ToStatus, &e.ChangedBy, &e.ChangedByName, &e.Note, &e.ChangedAt); err != nil {
			return nil, fmt.Errorf("scanning status history row: %w", err)
		}
		history = append(history, e)

	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating status history rows: %w", err)
	}

	return history, nil

}

func getCurrentPOStatus(ctx context.Context, tx pgx.Tx, poNumber string) (string, error) {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM finance_purchase_orders WHERE po_number = $1`, poNumber).Scan(&status)

	if err != nil {
		return "", fmt.Errorf("reading current status :%w", err)
	}
	return status, nil

}

func insertStatusHistory(ctx context.Context, tx pgx.Tx, poNumber, fromStatus, toStatus, changedBy string, note *string) error {
	_, err := tx.Exec(ctx, `INSERT INTO finance_order_status_history (po_number, from_status, to_status, changed_by, note)
		VALUES ($1, $2, $3, $4, $5)`, poNumber, fromStatus, toStatus, changedBy, note)

	if err != nil {
		return fmt.Errorf("inserting status history: %w", err)
	}
	return nil
}

var ErrLineItemsNotFullyVerified = errors.New("all line items must be verified before approval")

func ApprovePurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber string, verifiedLineItemIDs []int64, comment *string, approverID string) (*PurchaseOrderDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	currentStatus, err := getCurrentPOStatus(ctx, tx, poNumber)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `SELECT line_item_id FROM finance_order_line_items WHERE po_number = $1`, poNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching line items for verification: %w", err)
	}
	var allLineItemIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning line item id: %w", err)
		}
		allLineItemIDs = append(allLineItemIDs, id)
	}
	rows.Close()

	verifiedSet := make(map[int64]bool)
	for _, id := range verifiedLineItemIDs {
		verifiedSet[id] = true
	}
	for _, id := range allLineItemIDs {
		if !verifiedSet[id] {
			return nil, ErrLineItemsNotFullyVerified
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE finance_purchase_orders
		SET status = 'Approved', payment_status = 'Payment Ready', locked = true,
		    approver_id = $1, approval_date = CURRENT_DATE
		WHERE po_number = $2
	`, approverID, poNumber)
	if err != nil {
		return nil, fmt.Errorf("approving purchase order: %w", err)
	}

	if err := insertStatusHistory(ctx, tx, poNumber, currentStatus, "Approved", approverID, comment); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return GetPurchaseOrderDetail(ctx, pool, poNumber)
}

func RejectPurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber, reason, changedBy string) (*PurchaseOrderDetail, error) {
	return simpleStatusTransition(ctx, pool, poNumber, "Rejected", changedBy, &reason, nil)
}

func HoldPurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber, changedBy string, comment *string) (*PurchaseOrderDetail, error) {
	return simpleStatusTransition(ctx, pool, poNumber, "Hold", changedBy, comment, nil)
}

func RequestChangesPurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber, changedBy string, comment *string) (*PurchaseOrderDetail, error) {
	return simpleStatusTransition(ctx, pool, poNumber, "Received", changedBy, comment, nil)
}

func MarkPaidPurchaseOrder(ctx context.Context, pool *pgxpool.Pool, poNumber, changedBy string, comment *string) (*PurchaseOrderDetail, error) {
	extra := "UPDATE finance_purchase_orders SET payment_status = 'Paid' WHERE po_number = $1"
	return simpleStatusTransition(ctx, pool, poNumber, "Paid", changedBy, comment, &extra)
}

func simpleStatusTransition(ctx context.Context, pool *pgxpool.Pool, poNumber, newStatus, changedBy string, note *string, extraSQL *string) (*PurchaseOrderDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	currentStatus, err := getCurrentPOStatus(ctx, tx, poNumber)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE finance_purchase_orders SET status = $1 WHERE po_number = $2`, newStatus, poNumber)
	if err != nil {
		return nil, fmt.Errorf("updating status to %s: %w", newStatus, err)
	}

	if extraSQL != nil {
		if _, err := tx.Exec(ctx, *extraSQL, poNumber); err != nil {
			return nil, fmt.Errorf("applying extra update for %s: %w", newStatus, err)
		}
	}

	if err := insertStatusHistory(ctx, tx, poNumber, currentStatus, newStatus, changedBy, note); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return GetPurchaseOrderDetail(ctx, pool, poNumber)
}

func getPONumberForLineItem(ctx context.Context, tx pgx.Tx, lineItemID int64) (string, error) {
	var poNumber string
	err := tx.QueryRow(ctx, `SELECT po_number FROM finance_order_line_items WHERE line_item_id = $1`, lineItemID).Scan(&poNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLineItemNotFound
	}
	if err != nil {
		return "", fmt.Errorf("looking up po_number for line item: %w", err)
	}
	return poNumber, nil
}

var ErrLineItemNotFound = errors.New("line item not found")
var ErrCannotDeleteLastLineItem = errors.New("cannot delete the last remaining line item on a purchase order")

func GetLineItems(ctx context.Context, pool *pgxpool.Pool, poNumber string) ([]LineItemDetail, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_purchase_orders WHERE po_number = $1)`, poNumber).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking purchase order existence: %w", err)
	}
	if !exists {
		return nil, ErrPurchaseOrderNotFound
	}

	detail, err := GetPurchaseOrderDetail(ctx, pool, poNumber)
	if err != nil {
		return nil, err
	}
	return detail.LineItems, nil
}

func AddLineItem(ctx context.Context, pool *pgxpool.Pool, poNumber string, sku CreateSKUInput) (*LineItemDetail, error) {

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_purchase_orders WHERE po_number = $1)`, poNumber).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking purchase order existence: %w", err)
	}
	if !exists {
		return nil, ErrPurchaseOrderNotFound
	}

	var lineItemID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO finance_order_line_items (po_number, sku_code, product_name, quantity, rate_per_unit, packaging_flat, selling_price_per_unit)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING line_item_id
	`, poNumber, sku.SKUCode, sku.ProductName, sku.Quantity, sku.RatePerUnit, sku.PackagingFlat, sku.SellingPricePerUnit).Scan(&lineItemID)
	if err != nil {
		return nil, fmt.Errorf("inserting line item: %w", err)
	}

	for _, ch := range sku.Charges {
		_, err = tx.Exec(ctx, `
			INSERT INTO finance_line_item_charges (line_item_id, charge_type_id, rate_per_piece)
			VALUES ($1, $2, $3)
		`, lineItemID, ch.ChargeTypeID, ch.RatePerPiece)
		if err != nil {
			return nil, fmt.Errorf("inserting charge: %w", err)
		}
	}

	if err := recomputePOTotals(ctx, tx, poNumber); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return getLineItemDetail(ctx, pool, lineItemID)

}

type UpdateLineItemInput struct {
	SKUCode             *string
	ProductName         *string
	Quantity            *int
	RatePerUnit         *float64
	PackagingFlat       *float64
	SellingPricePerUnit *float64
}

func UpdateLineItem(ctx context.Context, pool *pgxpool.Pool, lineItemID int64, input UpdateLineItemInput) (*LineItemDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	poNumber, err := getPONumberForLineItem(ctx, tx, lineItemID)
	if err != nil {
		return nil, err
	}

	var setClauses []string
	var args []interface{}
	argPos := 1
	addField := func(column string, value interface{}) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, argPos))
		args = append(args, value)
		argPos++
	}

	if input.SKUCode != nil {
		addField("sku_code", *input.SKUCode)
	}
	if input.ProductName != nil {
		addField("product_name", *input.ProductName)
	}
	if input.Quantity != nil {
		addField("quantity", *input.Quantity)
	}
	if input.RatePerUnit != nil {
		addField("rate_per_unit", *input.RatePerUnit)
	}
	if input.PackagingFlat != nil {
		addField("packaging_flat", *input.PackagingFlat)
	}
	if input.SellingPricePerUnit != nil {
		addField("selling_price_per_unit", *input.SellingPricePerUnit)
	}

	if len(setClauses) > 0 {
		query := fmt.Sprintf(`UPDATE finance_order_line_items SET %s WHERE line_item_id = $%d`,
			strings.Join(setClauses, ", "), argPos)
		args = append(args, lineItemID)
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("updating line item: %w", err)
		}
	}

	if err := recomputePOTotals(ctx, tx, poNumber); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return getLineItemDetail(ctx, pool, lineItemID)
}

func DeleteLineItem(ctx context.Context, pool *pgxpool.Pool, lineItemID int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	poNumber, err := getPONumberForLineItem(ctx, tx, lineItemID)
	if err != nil {
		return err
	}

	var count int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM finance_order_line_items WHERE po_number = $1`, poNumber).Scan(&count)
	if err != nil {
		return fmt.Errorf("counting line items: %w", err)
	}
	if count <= 1 {
		return ErrCannotDeleteLastLineItem
	}

	_, err = tx.Exec(ctx, `DELETE FROM finance_order_line_items WHERE line_item_id = $1`, lineItemID)
	if err != nil {
		return fmt.Errorf("deleting line item: %w", err)
	}

	if err := recomputePOTotals(ctx, tx, poNumber); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func getLineItemDetail(ctx context.Context, pool *pgxpool.Pool, lineItemID int64) (*LineItemDetail, error) {
	var li LineItemDetail
	err := pool.QueryRow(ctx, `
		SELECT line_item_id, sku_code, product_name, quantity, rate_per_unit, packaging_flat, selling_price_per_unit
		FROM finance_order_line_items
		WHERE line_item_id = $1
	`, lineItemID).Scan(&li.LineItemID, &li.SKUCode, &li.ProductName, &li.Quantity, &li.RatePerUnit, &li.PackagingFlat, &li.SellingPricePerUnit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLineItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching line item: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT c.charge_id, c.charge_type_id, ct.name, c.rate_per_piece
		FROM finance_line_item_charges c
		JOIN finance_charge_types ct ON ct.charge_type_id = c.charge_type_id
		WHERE c.line_item_id = $1
	`, lineItemID)
	if err != nil {
		return nil, fmt.Errorf("fetching charges: %w", err)
	}
	defer rows.Close()

	li.Charges = []ChargeDetail{}
	for rows.Next() {
		var ch ChargeDetail
		if err := rows.Scan(&ch.ChargeID, &ch.ChargeTypeID, &ch.ChargeName, &ch.RatePerPiece); err != nil {
			return nil, fmt.Errorf("scanning charge: %w", err)
		}
		li.Charges = append(li.Charges, ch)
	}

	qty := float64(li.Quantity)
	var chargesPerPiece float64
	for _, ch := range li.Charges {
		chargesPerPiece += ch.RatePerPiece
	}
	li.ComputedLineTotal = qty*li.RatePerUnit + qty*li.PackagingFlat + qty*chargesPerPiece

	return &li, nil
}

func GetLineItemCharges(ctx context.Context, pool *pgxpool.Pool, lineItemID int64) ([]ChargeDetail, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_order_line_items WHERE line_item_id = $1)`, lineItemID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking line item existence: %w", err)
	}
	if !exists {
		return nil, ErrLineItemNotFound
	}

	rows, err := pool.Query(ctx, `
		SELECT c.charge_id, c.charge_type_id, ct.name, c.rate_per_piece
		FROM finance_line_item_charges c
		JOIN finance_charge_types ct ON ct.charge_type_id = c.charge_type_id
		WHERE c.line_item_id = $1
	`, lineItemID)
	if err != nil {
		return nil, fmt.Errorf("querying charges: %w", err)
	}
	defer rows.Close()

	charges := []ChargeDetail{}
	for rows.Next() {
		var ch ChargeDetail
		if err := rows.Scan(&ch.ChargeID, &ch.ChargeTypeID, &ch.ChargeName, &ch.RatePerPiece); err != nil {
			return nil, fmt.Errorf("scanning charge: %w", err)
		}
		charges = append(charges, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating charges: %w", err)
	}

	return charges, nil
}

func ReplaceLineItemCharges(ctx context.Context, pool *pgxpool.Pool, lineItemID int64, newCharges []CreateChargeInput) ([]ChargeDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	poNumber, err := getPONumberForLineItem(ctx, tx, lineItemID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `DELETE FROM finance_line_item_charges WHERE line_item_id = $1`, lineItemID)
	if err != nil {
		return nil, fmt.Errorf("clearing existing charges: %w", err)
	}

	for _, ch := range newCharges {
		_, err = tx.Exec(ctx, `
			INSERT INTO finance_line_item_charges (line_item_id, charge_type_id, rate_per_piece)
			VALUES ($1, $2, $3)
		`, lineItemID, ch.ChargeTypeID, ch.RatePerPiece)
		if err != nil {
			return nil, fmt.Errorf("inserting charge: %w", err)
		}
	}

	if err := recomputePOTotals(ctx, tx, poNumber); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return GetLineItemCharges(ctx, pool, lineItemID)
}

var ErrChargeNotFound = errors.New("charge not found")

func DeleteLineItemCharge(ctx context.Context, pool *pgxpool.Pool, lineItemID, chargeID int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	poNumber, err := getPONumberForLineItem(ctx, tx, lineItemID)
	if err != nil {
		return err
	}

	result, err := tx.Exec(ctx, `DELETE FROM finance_line_item_charges WHERE charge_id = $1 AND line_item_id = $2`, chargeID, lineItemID)
	if err != nil {
		return fmt.Errorf("deleting charge: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrChargeNotFound
	}

	if err := recomputePOTotals(ctx, tx, poNumber); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
