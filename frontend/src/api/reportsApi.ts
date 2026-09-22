import { apiFetch } from "./httpClient";

export const REPORT_TYPES = [
  "daily", "weekly", "monthly", "vendor", "margin", "gst", "payment", "procurement", "inventory-landing-cost",
] as const;

export type ReportType = (typeof REPORT_TYPES)[number];
export type ReportFormat = "csv" | "pdf";

export interface ReportFilters {
  vendorId?: string;
  from?: string;
  to?: string;
}

export async function exportReport(
  type: ReportType,
  format: ReportFormat,
  filters: ReportFilters = {}
): Promise<void> {
  const params = new URLSearchParams();
  params.set("type", type);
  params.set("format", format);
  if (filters.vendorId) params.set("vendorId", filters.vendorId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);

  const response = await apiFetch(`/reports/export?${params.toString()}`);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to export report");
  }

  const blob = await response.blob();
  const disposition = response.headers.get("Content-Disposition");
  const match = disposition?.match(/filename="?([^"]+)"?/);
  const filename = match?.[1] ?? `report_${type}.${format}`;

  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}