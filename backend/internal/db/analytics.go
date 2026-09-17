package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsKPIs struct {
	TotalOrders        int     `json:"total_orders"`
	TotalSpend          float64 `json:"total_spend"`
	TotalGST             float64 `json:"total_gst"`
	TotalBaseCost        float64 `json:"total_base_cost"`
	TotalPackagingCost   float64 `json:"total_packaging_cost"`
	TotalTransportCost   float64 `json:"total_transport_cost"`
	TotalHandlingCost    float64 `json:"total_handling_cost"`
	TotalOtherCharges    float64 `json:"total_other_charges"`
	AvgMarginPct         float64 `json:"avg_margin_pct"`
	AvgLandingPct        float64 `json:"avg_landing_pct"`
	TotalQtyProcured     int     `json:"total_qty_procured"`
	UniqueVendorsUsed    int     `json:"unique_vendors_used"`
	AvgLandingCostPerUnit float64 `json:"avg_landing_cost_per_unit"`
}

type AnalyticsOverview struct {
	KPIs                  AnalyticsKPIs `json:"kpis"`
	SpendByPaymentStatus  []ChartPoint  `json:"spend_by_payment_status"`
	SpendByOrderStatus    []ChartPoint  `json:"spend_by_order_status"`
	GSTByMonth            []ChartPoint  `json:"gst_by_month"`
	ProcurementByMonth    []ChartPoint  `json:"procurement_by_month"`
	MarginByVendor        []ChartPoint  `json:"margin_by_vendor"`
	PaymentStatusBreakdown []ChartPoint `json:"payment_status_breakdown"`
	ChargeTypeBreakdown   []ChartPoint  `json:"charge_type_breakdown"`
}

func GetAnalyticsOverview(ctx context.Context, pool *pgxpool.Pool, filters DashboardFilters) (*AnalyticsOverview, error) {
	var a AnalyticsOverview

	argPos := 1
	var args []interface{}
	whereClause := dashboardDateFilter(filters, &argPos, &args)

	kpiQuery := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(landing_cost), 0),
			COALESCE(SUM(gst_amount), 0),
			COALESCE(SUM(base_total), 0),
			COALESCE(SUM(packaging_total), 0),
			COALESCE(SUM(transport_total), 0),
			COALESCE(SUM(handling_total), 0),
			COALESCE(SUM(other_charges_total), 0),
			COALESCE(AVG(gross_margin_pct), 0),
			COALESCE(AVG(landing_pct), 0),
			COALESCE(SUM(total_qty), 0),
			COUNT(DISTINCT vendor_id),
			COALESCE(AVG(landing_cost_per_unit), 0)
		FROM finance_purchase_orders
		%s
	`, whereClause)

	err := pool.QueryRow(ctx, kpiQuery, args...).Scan(
		&a.KPIs.TotalOrders, &a.KPIs.TotalSpend, &a.KPIs.TotalGST, &a.KPIs.TotalBaseCost,
		&a.KPIs.TotalPackagingCost, &a.KPIs.TotalTransportCost, &a.KPIs.TotalHandlingCost,
		&a.KPIs.TotalOtherCharges, &a.KPIs.AvgMarginPct, &a.KPIs.AvgLandingPct,
		&a.KPIs.TotalQtyProcured, &a.KPIs.UniqueVendorsUsed, &a.KPIs.AvgLandingCostPerUnit,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching analytics KPIs: %w", err)
	}

	a.SpendByPaymentStatus, err = fetchGroupedChart(ctx, pool, "payment_status", "SUM(landing_cost)")
	if err != nil {
		return nil, fmt.Errorf("spend_by_payment_status: %w", err)
	}

	a.SpendByOrderStatus, err = fetchGroupedChart(ctx, pool, "status", "SUM(landing_cost)")
	if err != nil {
		return nil, fmt.Errorf("spend_by_order_status: %w", err)
	}

	a.PaymentStatusBreakdown, err = fetchGroupedChart(ctx, pool, "payment_status", "COUNT(*)")
	if err != nil {
		return nil, fmt.Errorf("payment_status_breakdown: %w", err)
	}

	a.GSTByMonth, err = fetchChartByMonth(ctx, pool, "SUM(gst_amount)", "", nil)
	if err != nil {
		return nil, fmt.Errorf("gst_by_month: %w", err)
	}

	a.ProcurementByMonth, err = fetchChartByMonth(ctx, pool, "SUM(total_qty)", "", nil)
	if err != nil {
		return nil, fmt.Errorf("procurement_by_month: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT v.name, COALESCE(AVG(po.gross_margin_pct), 0)
		FROM finance_vendors v
		LEFT JOIN finance_purchase_orders po ON po.vendor_id = v.vendor_id
		GROUP BY v.name
		ORDER BY v.name
	`)
	if err != nil {
		return nil, fmt.Errorf("margin_by_vendor: %w", err)
	}
	for rows.Next() {
		var cp ChartPoint
		if err := rows.Scan(&cp.Label, &cp.Value); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning margin by vendor: %w", err)
		}
		a.MarginByVendor = append(a.MarginByVendor, cp)
	}
	rows.Close()

	rows, err = pool.Query(ctx, `
		SELECT ct.name, COALESCE(SUM(c.rate_per_piece * li.quantity), 0)
		FROM finance_charge_types ct
		LEFT JOIN finance_line_item_charges c ON c.charge_type_id = ct.charge_type_id
		LEFT JOIN finance_order_line_items li ON li.line_item_id = c.line_item_id
		GROUP BY ct.name
		ORDER BY ct.name
	`)
	if err != nil {
		return nil, fmt.Errorf("charge_type_breakdown: %w", err)
	}
	for rows.Next() {
		var cp ChartPoint
		if err := rows.Scan(&cp.Label, &cp.Value); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning charge type breakdown: %w", err)
		}
		a.ChargeTypeBreakdown = append(a.ChargeTypeBreakdown, cp)
	}
	rows.Close()

	return &a, nil
}

func fetchGroupedChart(ctx context.Context, pool *pgxpool.Pool, groupCol, aggExpr string) ([]ChartPoint, error) {
	query := fmt.Sprintf(`
		SELECT %s, COALESCE(%s, 0)
		FROM finance_purchase_orders
		GROUP BY %s
		ORDER BY %s
	`, groupCol, aggExpr, groupCol, groupCol)

	rows, err := pool.Query(ctx, query)
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
