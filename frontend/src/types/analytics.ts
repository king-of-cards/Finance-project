import type { ChartPoint } from "./dashboard";

export interface AnalyticsKPIs {
  total_orders: number;
  total_spend: number;
  total_gst: number;
  total_base_cost: number;
  total_packaging_cost: number;
  total_transport_cost: number;
  total_handling_cost: number;
  total_other_charges: number;
  avg_margin_pct: number;
  avg_landing_pct: number;
  total_qty_procured: number;
  unique_vendors_used: number;
  avg_landing_cost_per_unit: number;
}

export interface AnalyticsOverview {
  kpis: AnalyticsKPIs;
  spend_by_payment_status: ChartPoint[];
  spend_by_order_status: ChartPoint[];
  gst_by_month: ChartPoint[];
  procurement_by_month: ChartPoint[];
  margin_by_vendor: ChartPoint[];
  payment_status_breakdown: ChartPoint[];
  charge_type_breakdown: ChartPoint[];
}