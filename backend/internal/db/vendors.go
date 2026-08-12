package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Vendor struct {
	VendorID        string    `json:"vendor_id"`
	Name            string    `json:"name"`
	City            *string   `json:"city"`
	ContactPerson   *string   `json:"contact_person"`
	Phone           *string   `json:"phone"`
	Email           *string   `json:"email"`
	Address         *string   `json:"address"`
	GSTNumber       *string   `json:"gst_number"`
	PANNumber       *string   `json:"pan_number"`
	PaymentTerms    string    `json:"payment_terms"`
	AvgDeliveryDays int       `json:"avg_delivery_days"`
	Rating          *int16    `json:"rating"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type VendorDetail struct {
	Vendor
	TotalProcurement float64  `json:"total_procurement"`
	AvgMargin        float64  `json:"avg_margin"`
	OrdersCompleted  int      `json:"orders_completed"`
	PendingOrders    int      `json:"pending_orders"`
	AvgDelivery      *float64 `json:"avg_delivery"`
}

type VendorFilters struct {
	Status string
	Search string
}

type VendorOrder struct {
	PONumber             string     `json:"po_number"`
	Status               string     `json:"status"`
	PaymentStatus        string     `json:"payment_status"`
	OrderedDate          *time.Time `json:"ordered_date"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`
	ReceivedDate         *time.Time `json:"received_date"`
	LandingCost          float64    `json:"landing_cost"`
	GrossMarginPct       float64    `json:"gross_margin_pct"`
	TotalQty             int        `json:"total_qty"`
}

type PaginatedVendorOrders struct {
	Orders     []VendorOrder `json:"orders"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	Total      int           `json:"total"`
	TotalPages int           `json:"total_pages"`
}

var ErrVendorNotFound = errors.New("vendor not found ")

// buildVendorQuery is pure — no DB access — so it's unit testable in isolation.
func buildVendorQuery(filters VendorFilters) (string, []interface{}) {
	query := `SELECT vendor_id, name, city, contact_person, phone, email, address,
		       gst_number, pan_number, payment_terms, avg_delivery_days, rating,
		       status, created_at
		FROM finance_vendors`

	var conditions []string
	var args []interface{}
	argPos := 1

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filters.Status)
		argPos++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argPos))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	return query, args
}

func GetVendors(ctx context.Context, pool *pgxpool.Pool, filters VendorFilters) ([]Vendor, error) {
	query, args := buildVendorQuery(filters)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying finance_vendors: %w", err)
	}
	defer rows.Close()

	var vendors []Vendor
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(
			&v.VendorID, &v.Name, &v.City, &v.ContactPerson, &v.Phone, &v.Email,
			&v.Address, &v.GSTNumber, &v.PANNumber, &v.PaymentTerms,
			&v.AvgDeliveryDays, &v.Rating, &v.Status, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning vendor row: %w", err)
		}
		vendors = append(vendors, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating vendor rows: %w", err)
	}
	return vendors, nil
}

func GetVendorDetail(ctx context.Context, pool *pgxpool.Pool, vendorID string) (*VendorDetail, error) {
	var d VendorDetail

	err := pool.QueryRow(ctx, `
		SELECT vendor_id, name, city, contact_person, phone, email, address,
		       gst_number, pan_number, payment_terms, avg_delivery_days, rating,
		       status, created_at
		FROM finance_vendors
		WHERE vendor_id = $1
	`, vendorID).Scan(
		&d.VendorID, &d.Name, &d.City, &d.ContactPerson, &d.Phone, &d.Email,
		&d.Address, &d.GSTNumber, &d.PANNumber, &d.PaymentTerms,
		&d.AvgDeliveryDays, &d.Rating, &d.Status, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVendorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("quering vendor: %w", err)
	}

	err = pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(landing_cost), 0) AS total_procurement,
			COALESCE(AVG(gross_margin_pct), 0) AS avg_margin,
			COUNT(*) FILTER (WHERE status IN ('Received','Verified','Approved','Paid')) AS orders_completed,
			COUNT(*) FILTER (WHERE status IN ('PO Raised','Waiting for Vendor','Pending Approval')) AS pending_orders,
			AVG(received_date - ordered_date) FILTER (WHERE received_date IS NOT 
			 NULL AND ordered_date IS NOT NULL) AS avg_delivery
		FROM finance_purchase_orders
		WHERE vendor_id = $1
	`, vendorID).Scan(
		&d.TotalProcurement, &d.AvgMargin, &d.OrdersCompleted, &d.PendingOrders, &d.AvgDelivery,
	)
	if err != nil {
		return nil, fmt.Errorf("quering vendor rollups: %w", err)
	}

	return &d, nil

}

func GetVendorOrders(ctx context.Context, pool *pgxpool.Pool, vendorID string, page, limit int) (*PaginatedVendorOrders, error) {
	var total int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM finance_purchase_orders WHERE vendor_id = $1
	`, vendorID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("counting vendor orders: %w", err)
	}

	offset := (page - 1) * limit

	rows, err := pool.Query(ctx, `
		SELECT po_number, status, payment_status, ordered_date, expected_delivery_date,
		       received_date, landing_cost, gross_margin_pct, total_qty
		FROM finance_purchase_orders
		WHERE vendor_id = $1
		ORDER BY ordered_date DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`, vendorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("querying vendor orders: %w", err)
	}
	defer rows.Close()

	var orders []VendorOrder
	for rows.Next() {
		var o VendorOrder
		if err := rows.Scan(
			&o.PONumber, &o.Status, &o.PaymentStatus, &o.OrderedDate, &o.ExpectedDeliveryDate,
			&o.ReceivedDate, &o.LandingCost, &o.GrossMarginPct, &o.TotalQty,
		); err != nil {
			return nil, fmt.Errorf("scanning vendor order row: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating vendor order rows: %w", err)
	}

	totalPages := (total + limit - 1) / limit // ceiling division
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedVendorOrders{
		Orders:     orders,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

type CreateVendorInput struct {
	Name          string
	City          *string
	ContactPerson *string
	Phone         *string
	Email         *string
	Address       *string
	GSTNumber     *string
	PANNumber     *string
	PaymentTerms  string
}

func generateNextVendorID(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var maxNum int

	err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(CAST(SUBSTRING(vendor_id FROM 2) AS INTEGER)), 0)
		FROM finance_vendors
		WHERE vendor_id ~ '^V[0-9]+$'
	`).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("generating next vendor ID: %w", err)
	}
	return fmt.Sprintf("V%03d", maxNum+1), nil

}

func CreateVendor(ctx context.Context, pool *pgxpool.Pool, input CreateVendorInput) (*Vendor, error) {
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		vendorID, err := generateNextVendorID(ctx, pool)
		if err != nil {
			return nil, err
		}

		var v Vendor

		query := `
			INSERT INTO finance_vendors
				(vendor_id, name, city, contact_person, phone, email, address, gst_number, pan_number, payment_terms)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, COALESCE(NULLIF($10, ''), 'Net 30'))
			RETURNING vendor_id, name, city, contact_person, phone, email, address,
			          gst_number, pan_number, payment_terms, avg_delivery_days, rating, status, created_at
		`

		err = pool.QueryRow(ctx, query, vendorID, input.Name, input.City, input.ContactPerson, input.Phone, input.Email,
			input.Address, input.GSTNumber, input.PANNumber, input.PaymentTerms,
		).Scan(
			&v.VendorID, &v.Name, &v.City, &v.ContactPerson, &v.Phone, &v.Email,
			&v.Address, &v.GSTNumber, &v.PANNumber, &v.PaymentTerms,
			&v.AvgDeliveryDays, &v.Rating, &v.Status, &v.CreatedAt,
		)
		if err == nil {
			return &v, nil
		}

		if strings.Contains(err.Error(), "duplicate key") {
			continue
		}

		return nil, fmt.Errorf("creating vendor: %w", err)
	}

	return nil, fmt.Errorf("creating vendor: exhausted retries generating a unique vendor_id")
}

type UpdateVendorInput struct {
	Name          *string
	City          *string
	ContactPerson *string
	Phone         *string
	Email         *string
	Address       *string
	GSTNumber     *string
	PANNumber     *string
	PaymentTerms  *string
	Status        *string
	Rating        *int16
}

var ErrNoFieldsToUpdate = errors.New("no fields provided to update")

func UpdateVendor(ctx context.Context, pool *pgxpool.Pool, vendorID string, input UpdateVendorInput) (*Vendor, error) {
	var setClauses []string
	var args []interface{}
	argPos := 1

	addField := func(column string, value interface{}) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, argPos))
		args = append(args, value)
		argPos++
	}

	if input.Name != nil {
		addField("name", *input.Name)
	}
	if input.City != nil {
		addField("city", *input.City)
	}
	if input.ContactPerson != nil {
		addField("contact_person", *input.ContactPerson)
	}
	if input.Phone != nil {
		addField("phone", *input.Phone)
	}
	if input.Email != nil {
		addField("email", *input.Email)
	}
	if input.Address != nil {
		addField("address", *input.Address)
	}
	if input.GSTNumber != nil {
		addField("gst_number", *input.GSTNumber)
	}
	if input.PANNumber != nil {
		addField("pan_number", *input.PANNumber)
	}
	if input.PaymentTerms != nil {
		addField("payment_terms", *input.PaymentTerms)
	}
	if input.Status != nil {
		addField("status", *input.Status)
	}
	if input.Rating != nil {
		addField("rating", *input.Rating)
	}

	if len(setClauses) == 0 {
		return nil, ErrNoFieldsToUpdate
	}
	query := fmt.Sprintf(`
		UPDATE finance_vendors
		SET %s
		WHERE vendor_id = $%d
		RETURNING vendor_id, name, city, contact_person, phone, email, address,
		          gst_number, pan_number, payment_terms, avg_delivery_days, rating, status, created_at
	`, strings.Join(setClauses, ", "), argPos)

	args = append(args, vendorID)
	var v Vendor
	err := pool.QueryRow(ctx, query, args...).Scan(
		&v.VendorID, &v.Name, &v.City, &v.ContactPerson, &v.Phone, &v.Email,
		&v.Address, &v.GSTNumber, &v.PANNumber, &v.PaymentTerms,
		&v.AvgDeliveryDays, &v.Rating, &v.Status, &v.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVendorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating vendor: %w", err)
	}

	return &v, nil
}

func DeactivateVendor(ctx context.Context, pool *pgxpool.Pool, vendorID string) (*Vendor, error) {
	inactive := "Inactive"
	return UpdateVendor(ctx, pool, vendorID, UpdateVendorInput{
		Status: &inactive,
	})
}
