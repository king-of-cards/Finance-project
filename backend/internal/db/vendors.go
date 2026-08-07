package db

import (
	"context"
	"fmt"
	"strings"
	"time"

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

type VendorFilters struct {
	Status string
	Search string
}

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
