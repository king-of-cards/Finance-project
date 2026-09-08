import type {
  CreatePurchaseOrderRequest,
  UpdatePurchaseOrderRequest,
  PaginatedPurchaseOrders,
  PurchaseOrderDetail,
  PurchaseOrderFilters,
} from "../types/purchaseOrder";
import { apiFetch, apiJson } from "./httpClient";

function buildQuery(
  filters: PurchaseOrderFilters,
  extra?: Record<string, string | number>
) {
  const params = new URLSearchParams();

  if (filters.status) params.set("status", filters.status);
  if (filters.paymentStatus) params.set("paymentStatus", filters.paymentStatus);
  if (filters.vendorId) params.set("vendorId", filters.vendorId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  if (filters.search) params.set("search", filters.search);

  if (extra) {
    for (const [key, value] of Object.entries(extra)) {
      params.set(key, String(value));
    }
  }

  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export async function getPurchaseOrders(
  filters: PurchaseOrderFilters,
  page: number,
  limit: number
): Promise<PaginatedPurchaseOrders> {
  const query = buildQuery(filters, { page, limit });

  return apiJson<PaginatedPurchaseOrders>(
    `/purchase-orders${query}`,
    {},
    "Failed to fetch purchase orders"
  );
}

export async function getPurchaseOrderDetail(
  poNumber: string
): Promise<PurchaseOrderDetail> {
  return apiJson<PurchaseOrderDetail>(
    `/purchase-orders/${encodeURIComponent(poNumber)}`,
    {},
    "Failed to fetch purchase order detail"
  );
}

export async function createPurchaseOrder(
  payload: CreatePurchaseOrderRequest
): Promise<PurchaseOrderDetail> {
  return apiJson<PurchaseOrderDetail>(
    "/purchase-orders",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    "Failed to create purchase order"
  );
}

export async function updatePurchaseOrder(
  poNumber: string,
  payload: UpdatePurchaseOrderRequest
): Promise<PurchaseOrderDetail> {
  return apiJson<PurchaseOrderDetail>(
    `/purchase-orders/${encodeURIComponent(poNumber)}`,
    {
      method: "PATCH",
      body: JSON.stringify(payload),
    },
    "Failed to update purchase order"
  );
}

export async function exportPurchaseOrders(
  filters: PurchaseOrderFilters
): Promise<void> {
  const query = buildQuery(filters);

  const response = await apiFetch(`/purchase-orders/export${query}`);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to export purchase orders");
  }

  const blob = await response.blob();
  const disposition = response.headers.get("Content-Disposition");
  const match = disposition?.match(/filename="?([^"]+)"?/);
  const filename = match?.[1] ?? "purchase_orders.csv";

  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");

  link.href = url;
  link.download = filename;

  document.body.appendChild(link);
  link.click();

  link.remove();
  window.URL.revokeObjectURL(url);
}