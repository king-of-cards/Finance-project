package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardFilters struct {
	VendorID string
	From     string
	To       string
}

type KPIs struct {
	TotalOrders      int     `json:"total_orders"`
	TotalSpend       float64 `json:"total_spend"`
	PendingApprovals int     `json:"pending_approvals"`
	ActiveVendors    int     `json:"active_vendors"`
	AvgMarginPct     float64 `json:"avg_margin_pct"`
	OverdueOrders    int     `json:"overdue_orders"`
	TotalPaid        float64 `json:"total_paid"`
	AvgDeliveryDays  float64 `json:"avg_delivery_days"`
}

type ChartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type DashboardOverview struct {
	KPIs                 KPIs                 `json:"kpis"`
	MonthlyValue         []ChartPoint         `json:"monthly_value"`
	VendorWiseSpend      []ChartPoint         `json:"vendor_wise_spend"`
	LandingCostTrend     []ChartPoint         `json:"landing_cost_trend"`
	MarginTrend          []ChartPoint         `json:"margin_trend"`
	PendingApprovalTrend []ChartPoint         `json:"pending_approval_trend"`
	VendorPerformance    []ChartPoint         `json:"vendor_performance"`
	TransportTrend       []ChartPoint         `json:"transport_trend"`
	RecentOrders         []PurchaseOrder      `json:"recent_orders"`
	RecentStatusChanges  []StatusHistoryEntry `json:"recent_status_changes"`
	RecentComments       []Comment            `json:"recent_comments"`
}

func dashboardDateFilter(filters DashboardFilters, argPos *int, args *[]interface{}) string {
	var clauses []string
	if filters.VendorID != "" {
		clauses = append(clauses, fmt.Sprintf("vendor_id = $%d", *argPos))
		*args = append(*args, filters.VendorID)
		*argPos++
	}
	if filters.From != "" {
		clauses = append(clauses, fmt.Sprintf("ordered_date >= $%d", *argPos))
		*args = append(*args, filters.From)
		*argPos++
	}
	if filters.To != "" {
		clauses = append(clauses, fmt.Sprintf("ordered_date <= $%d", *argPos))
		*args = append(*args, filters.To)
		*argPos++
	}
	if len(clauses) == 0 {
		return ""
	}
	out := " WHERE "
	for i, c := range clauses {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}

func GetDashboardOverview(ctx context.Context, pool *pgxpool.Pool, filters DashboardFilters) (*DashboardOverview, error) {
	var d DashboardOverview

	argPos := 1
	var args []interface{}
	whereClause := dashboardDateFilter(filters, &argPos, &args)

	// --- KPIs ---
	kpiQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_orders,
			COALESCE(SUM(landing_cost), 0) AS total_spend,
			COUNT(*) FILTER (WHERE status IN ('PO Raised','Waiting for Vendor','Pending Approval')) AS pending_approvals,
			COALESCE(AVG(gross_margin_pct), 0) AS avg_margin_pct,
			COUNT(*) FILTER (WHERE expected_delivery_date < CURRENT_DATE AND received_date IS NULL) AS overdue_orders,
			COALESCE(SUM(landing_cost) FILTER (WHERE payment_status = 'Paid'), 0) AS total_paid,
			COALESCE(AVG(received_date - ordered_date) FILTER (WHERE received_date IS NOT NULL), 0) AS avg_delivery_days
		FROM finance_purchase_orders
		%s
	`, whereClause)

	err := pool.QueryRow(ctx, kpiQuery, args...).Scan(
		&d.KPIs.TotalOrders, &d.KPIs.TotalSpend, &d.KPIs.PendingApprovals,
		&d.KPIs.AvgMarginPct, &d.KPIs.OverdueOrders, &d.KPIs.TotalPaid, &d.KPIs.AvgDeliveryDays,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching KPIs: %w", err)
	}

	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM finance_vendors WHERE status = 'Active'`).Scan(&d.KPIs.ActiveVendors)
	if err != nil {
		return nil, fmt.Errorf("fetching active vendor count: %w", err)
	}

	// --- Charts ---
	monthlyArgPos, monthlyArgs := 1, []interface{}{}
	monthlyWhere := dashboardDateFilter(filters, &monthlyArgPos, &monthlyArgs)
	d.MonthlyValue, err = fetchChartByMonth(ctx, pool, "SUM(landing_cost)", monthlyWhere, monthlyArgs)
	if err != nil {
		return nil, fmt.Errorf("monthly_value: %w", err)
	}

	landingArgPos, landingArgs := 1, []interface{}{}
	landingWhere := dashboardDateFilter(filters, &landingArgPos, &landingArgs)
	d.LandingCostTrend, err = fetchChartByMonth(ctx, pool, "SUM(landing_cost)", landingWhere, landingArgs)
	if err != nil {
		return nil, fmt.Errorf("landing_cost_trend: %w", err)
	}

	marginArgPos, marginArgs := 1, []interface{}{}
	marginWhere := dashboardDateFilter(filters, &marginArgPos, &marginArgs)
	d.MarginTrend, err = fetchChartByMonth(ctx, pool, "AVG(gross_margin_pct)", marginWhere, marginArgs)
	if err != nil {
		return nil, fmt.Errorf("margin_trend: %w", err)
	}

	transportArgPos, transportArgs := 1, []interface{}{}
	transportWhere := dashboardDateFilter(filters, &transportArgPos, &transportArgs)
	d.TransportTrend, err = fetchChartByMonth(ctx, pool, "SUM(transport_total)", transportWhere, transportArgs)
	if err != nil {
		return nil, fmt.Errorf("transport_trend: %w", err)
	}

	pendingArgPos, pendingArgs := 1, []interface{}{}
	pendingWhereBase := dashboardDateFilter(filters, &pendingArgPos, &pendingArgs)
	pendingWhere := pendingWhereBase
	if pendingWhere == "" {
		pendingWhere = " WHERE status IN ('PO Raised','Waiting for Vendor','Pending Approval')"
	} else {
		pendingWhere += " AND status IN ('PO Raised','Waiting for Vendor','Pending Approval')"
	}
	d.PendingApprovalTrend, err = fetchChartByMonth(ctx, pool, "COUNT(*)", pendingWhere, pendingArgs)
	if err != nil {
		return nil, fmt.Errorf("pending_approval_trend: %w", err)
	}

	// rows, err := pool.Query(ctx, `
	// 	SELECT v.name, COALESCE(SUM(po.landing_cost), 0) AS spend
	// 	FROM finance_vendors v
	// 	LEFT JOIN finance_purchase_orders po ON po.vendor_id = v.vendor_id
	// 	GROUP BY v.name
	// 	ORDER BY spend DESC
	// 	LIMIT 10
	// `)

	vendorSpendArgPos, vendorSpendArgs := 1, []interface{}{}
	vendorSpendFilter := dashboardJoinFilter(filters, &vendorSpendArgPos, &vendorSpendArgs, "po")

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT v.name, COALESCE(SUM(po.landing_cost), 0) AS spend
		FROM finance_vendors v
		LEFT JOIN finance_purchase_orders po ON po.vendor_id = v.vendor_id %s
		GROUP BY v.name
		ORDER BY spend DESC
		LIMIT 10
	`, vendorSpendFilter), vendorSpendArgs...)
	if err != nil {
		return nil, fmt.Errorf("vendor_wise_spend: %w", err)
	}
	for rows.Next() {
		var cp ChartPoint
		if err := rows.Scan(&cp.Label, &cp.Value); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning vendor spend: %w", err)
		}
		d.VendorWiseSpend = append(d.VendorWiseSpend, cp)
	}
	rows.Close()

	// rows, err = pool.Query(ctx, `
	// 	SELECT v.name, COALESCE(AVG(po.received_date - po.ordered_date), 0) AS avg_days
	// 	FROM finance_vendors v
	// 	LEFT JOIN finance_purchase_orders po ON po.vendor_id = v.vendor_id AND po.received_date IS NOT NULL
	// 	GROUP BY v.name
	// 	ORDER BY avg_days ASC
	// 	LIMIT 10
	// `)

	vendorPerfArgPos, vendorPerfArgs := 1, []interface{}{}
	vendorPerfFilter := dashboardJoinFilter(filters, &vendorPerfArgPos, &vendorPerfArgs, "po")

	rows, err = pool.Query(ctx, fmt.Sprintf(`
		SELECT v.name, COALESCE(AVG(po.received_date - po.ordered_date), 0) AS avg_days
		FROM finance_vendors v
		LEFT JOIN finance_purchase_orders po ON po.vendor_id = v.vendor_id AND po.received_date IS NOT NULL %s
		GROUP BY v.name
		ORDER BY avg_days ASC
		LIMIT 10
	`, vendorPerfFilter), vendorPerfArgs...)
	if err != nil {
		return nil, fmt.Errorf("vendor_performance: %w", err)
	}
	for rows.Next() {
		var cp ChartPoint
		if err := rows.Scan(&cp.Label, &cp.Value); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning vendor performance: %w", err)
		}
		d.VendorPerformance = append(d.VendorPerformance, cp)
	}
	rows.Close()

	// --- Recent activity ---
	poRows, err := pool.Query(ctx, `
		SELECT po.po_number, po.customer_order_no, po.vendor_id, v.name, po.status, po.payment_status,
		       po.ordered_date, po.expected_delivery_date, po.invoice_no, po.landing_cost, po.gross_margin_pct, po.total_qty
		FROM finance_purchase_orders po
		JOIN finance_vendors v ON v.vendor_id = po.vendor_id
		ORDER BY po.created_date DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("recent_orders: %w", err)
	}
	for poRows.Next() {
		var o PurchaseOrder
		if err := poRows.Scan(&o.PONumber, &o.CustomerOrderNo, &o.VendorID, &o.VendorName, &o.Status, &o.PaymentStatus,
			&o.OrderedDate, &o.ExpectedDeliveryDate, &o.InvoiceNo, &o.LandingCost, &o.GrossMarginPct, &o.TotalQty); err != nil {
			poRows.Close()
			return nil, fmt.Errorf("scanning recent order: %w", err)
		}
		d.RecentOrders = append(d.RecentOrders, o)
	}
	poRows.Close()

	histRows, err := pool.Query(ctx, `
		SELECT h.history_id, h.from_status, h.to_status, h.changed_by, u.name, h.note, h.changed_at
		FROM finance_order_status_history h
		JOIN finance_users u ON u.user_id = h.changed_by
		ORDER BY h.changed_at DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("recent_status_changes: %w", err)
	}
	for histRows.Next() {
		var e StatusHistoryEntry
		if err := histRows.Scan(&e.HistoryID, &e.FromStatus, &e.ToStatus, &e.ChangedBy, &e.ChangedByName, &e.Note, &e.ChangedAt); err != nil {
			histRows.Close()
			return nil, fmt.Errorf("scanning status change: %w", err)
		}
		d.RecentStatusChanges = append(d.RecentStatusChanges, e)
	}
	histRows.Close()

	commentRows, err := pool.Query(ctx, `
		SELECT c.comment_id, c.po_number, c.author_id, u.name, c.comment_text, c.created_at
		FROM finance_order_comments c
		JOIN finance_users u ON u.user_id = c.author_id
		ORDER BY c.created_at DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("recent_comments: %w", err)
	}
	for commentRows.Next() {
		var cm Comment
		if err := commentRows.Scan(&cm.CommentID, &cm.PONumber, &cm.AuthorID, &cm.AuthorName, &cm.CommentText, &cm.CreatedAt); err != nil {
			commentRows.Close()
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		d.RecentComments = append(d.RecentComments, cm)
	}
	commentRows.Close()

	return &d, nil
}

// func fetchChartByMonth(ctx context.Context, pool *pgxpool.Pool, aggExpr, whereClause string, args []interface{}) ([]ChartPoint, error) {
// 	query := fmt.Sprintf(`
// 		SELECT TO_CHAR(ordered_date, 'YYYY-MM') AS month, COALESCE(%s, 0)
// 		FROM finance_purchase_orders
// 		%s
// 		GROUP BY month
// 		ORDER BY month
// 	`, aggExpr, whereClause)

// 	rows, err := pool.Query(ctx, query, args...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var points []ChartPoint
// 	for rows.Next() {
// 		var cp ChartPoint
// 		if err := rows.Scan(&cp.Label, &cp.Value); err != nil {
// 			return nil, err
// 		}
// 		points = append(points, cp)
// 	}
// 	return points, rows.Err()
// }

func fetchChartByMonth(ctx context.Context, pool *pgxpool.Pool, aggExpr, whereClause string, args []interface{}) ([]ChartPoint, error) {
	nullGuard := "ordered_date IS NOT NULL"
	if whereClause == "" {
		whereClause = " WHERE " + nullGuard
	} else {
		whereClause += " AND " + nullGuard
	}

	query := fmt.Sprintf(`
		SELECT TO_CHAR(ordered_date, 'YYYY-MM') AS month, COALESCE(%s, 0)
		FROM finance_purchase_orders
		%s
		GROUP BY month
		ORDER BY month
	`, aggExpr, whereClause)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []ChartPoint
	for rows.Next() {
		var cp ChartPoint
		if err := rows.Scan(&cp.Label, &cp.Value); err != nil {
			return nil, err
		}
		points = append(points, cp)
	}
	return points, rows.Err()
}

func dashboardJoinFilter(filters DashboardFilters, argPos *int, args *[]interface{}, alias string) string {
	var clauses []string
	if filters.VendorID != "" {
		clauses = append(clauses, fmt.Sprintf("%s.vendor_id = $%d", alias, *argPos))
		*args = append(*args, filters.VendorID)
		*argPos++
	}
	if filters.From != "" {
		clauses = append(clauses, fmt.Sprintf("%s.ordered_date >= $%d", alias, *argPos))
		*args = append(*args, filters.From)
		*argPos++
	}
	if filters.To != "" {
		clauses = append(clauses, fmt.Sprintf("%s.ordered_date <= $%d", alias, *argPos))
		*args = append(*args, filters.To)
		*argPos++
	}
	if len(clauses) == 0 {
		return ""
	}
	out := " AND "
	for i, c := range clauses {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}