import type { DashboardFilters, DashboardOverview } from "../types/dashboard";
import { apiJson } from "./httpClient";

function buildQuery(filters: DashboardFilters): string {
  const params = new URLSearchParams();
  if (filters.vendorId) params.set("vendorId", filters.vendorId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export async function getDashboardOverview(
  filters: DashboardFilters = {}
): Promise<DashboardOverview> {
  return apiJson<DashboardOverview>(
    `/dashboard/overview${buildQuery(filters)}`,
    {},
    "Failed to fetch dashboard overview"
  );
}