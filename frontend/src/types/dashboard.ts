import type { PurchaseOrder } from "./purchaseOrder";

export interface DashboardFilters {
  vendorId?: string;
  from?: string;
  to?: string;
}

export interface DashboardKPIs {
  total_orders: number;
  total_spend: number;
  pending_approvals: number;
  active_vendors: number;
  avg_margin_pct: number;
  overdue_orders: number;
  total_paid: number;
  avg_delivery_days: number;
}

export interface ChartPoint {
  label: string;
  value: number;
}

export interface StatusHistoryEntry {
  history_id: number;
  from_status: string | null;
  to_status: string;
  changed_by: string;
  changed_by_name: string;
  note: string | null;
  changed_at: string;
}

export interface DashboardOverview {
  kpis: DashboardKPIs;
  monthly_value: ChartPoint[];
  vendor_wise_spend: ChartPoint[];
  landing_cost_trend: ChartPoint[];
  margin_trend: ChartPoint[];
  pending_approval_trend: ChartPoint[];
  vendor_performance: ChartPoint[];
  transport_trend: ChartPoint[];
  recent_orders: PurchaseOrder[];
  recent_status_changes: StatusHistoryEntry[];
}