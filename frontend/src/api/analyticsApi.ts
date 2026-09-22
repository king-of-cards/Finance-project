import type { DashboardFilters } from "../types/dashboard";
import type { AnalyticsOverview } from "../types/analytics";
import { apiJson } from "./httpClient";

function buildQuery(filters: DashboardFilters): string {
  const params = new URLSearchParams();
  if (filters.vendorId) params.set("vendorId", filters.vendorId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

// Analytics reuses the same filter shape as the dashboard (vendorId/from/to).
export async function getAnalyticsOverview(
  filters: DashboardFilters = {}
): Promise<AnalyticsOverview> {
  return apiJson<AnalyticsOverview>(
    `/analytics/overview${buildQuery(filters)}`,
    {},
    "Failed to fetch analytics overview"
  );
}