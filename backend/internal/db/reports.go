package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRow []string

type ReportResult struct {
	Headers []string
	Rows    []ReportRow
}

var reportQueries = map[string]struct {
	Headers []string
	SQL     string
}{
	"daily": {
		Headers: []string{"Date", "Order Count", "Total Landing Cost"},
		SQL: `SELECT ordered_date::text, COUNT(*)::text, COALESCE(SUM(landing_cost),0)::text
		      FROM finance_purchase_orders %s GROUP BY ordered_date ORDER BY ordered_date`,
	},
	"weekly": {
		Headers: []string{"Week Starting", "Order Count", "Total Landing Cost"},
		SQL: `SELECT DATE_TRUNC('week', ordered_date)::date::text, COUNT(*)::text, COALESCE(SUM(landing_cost),0)::text
		      FROM finance_purchase_orders %s GROUP BY 1 ORDER BY 1`,
	},
	"monthly": {
		Headers: []string{"Month", "Order Count", "Total Landing Cost"},
		SQL: `SELECT TO_CHAR(ordered_date,'YYYY-MM'), COUNT(*)::text, COALESCE(SUM(landing_cost),0)::text
		      FROM finance_purchase_orders %s GROUP BY 1 ORDER BY 1`,
	},
	"vendor": {
		Headers: []string{"Vendor", "Order Count", "Total Spend", "Avg Margin %"},
		SQL: `SELECT v.name, COUNT(po.po_number)::text, COALESCE(SUM(po.landing_cost),0)::text, COALESCE(AVG(po.gross_margin_pct),0)::text
		      FROM finance_vendors v LEFT JOIN finance_purchase_orders po ON po.vendor_id = v.vendor_id %s GROUP BY v.name ORDER BY v.name`,
	},
	"margin": {
		Headers: []string{"PO Number", "Vendor", "Landing Cost", "Selling Total", "Gross Profit", "Margin %"},
		SQL: `SELECT po.po_number, v.name, po.landing_cost::text, po.selling_total::text, po.gross_profit::text, po.gross_margin_pct::text
		      FROM finance_purchase_orders po JOIN finance_vendors v ON v.vendor_id = po.vendor_id %s ORDER BY po.ordered_date DESC`,
	},
	"gst": {
		Headers: []string{"PO Number", "Vendor", "Base Total", "GST %", "GST Amount", "Landing Cost"},
		SQL: `SELECT po.po_number, v.name, po.base_total::text, po.gst_pct::text, po.gst_amount::text, po.landing_cost::text
		      FROM finance_purchase_orders po JOIN finance_vendors v ON v.vendor_id = po.vendor_id %s ORDER BY po.gst_pct, po.ordered_date DESC`,
	},
	"payment": {
		Headers: []string{"PO Number", "Vendor", "Payment Status", "Landing Cost", "Due Date"},
		SQL: `SELECT po.po_number, v.name, po.payment_status, po.landing_cost::text, COALESCE(po.due_date::text,'')
		      FROM finance_purchase_orders po JOIN finance_vendors v ON v.vendor_id = po.vendor_id
		      %s ORDER BY po.due_date NULLS LAST`,
	},
	"procurement": {
		Headers: []string{"PO Number", "Vendor", "Total Qty", "Landing Cost", "Landing Cost/Unit"},
		SQL: `SELECT po.po_number, v.name, po.total_qty::text, po.landing_cost::text, po.landing_cost_per_unit::text
		      FROM finance_purchase_orders po JOIN finance_vendors v ON v.vendor_id = po.vendor_id %s ORDER BY po.ordered_date DESC`,
	},
	"inventory-landing-cost": {
		Headers: []string{"SKU", "Product Name", "Quantity", "Rate/Unit", "Packaging/Unit"},
		SQL: `SELECT COALESCE(li.sku_code,''), li.product_name, li.quantity::text, li.rate_per_unit::text, li.packaging_flat::text
		      FROM finance_order_line_items li JOIN finance_purchase_orders po ON po.po_number = li.po_number %s ORDER BY li.product_name`,
	},
}

func GetReport(ctx context.Context, pool *pgxpool.Pool, reportType string, filters PurchaseOrderFilters) (*ReportResult, error) {
	spec, ok := reportQueries[reportType]
	if !ok {
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}

	whereClause := ""
	var args []interface{}
	argPos := 1
	var conditions []string

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

	if reportType == "payment" {
		conditions = append(conditions, "po.payment_status = 'Paid'")
	}

	if len(conditions) > 0 {
		whereClause = "WHERE " + joinAnd(conditions)
	}

	query := fmt.Sprintf(spec.SQL, whereClause)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying report %s: %w", reportType, err)
	}
	defer rows.Close()

	result := &ReportResult{Headers: spec.Headers}
	cols := len(spec.Headers)

	for rows.Next() {
		vals := make([]interface{}, cols)
		ptrs := make([]interface{}, cols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scanning report row: %w", err)
		}
		row := make(ReportRow, cols)
		for i, v := range vals {
			if v == nil {
				row[i] = ""
			} else if s, ok := v.(string); ok {
				row[i] = s
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating report rows: %w", err)
	}

	return result, nil
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}
