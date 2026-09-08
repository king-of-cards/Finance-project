export interface PurchaseOrder {
    po_number: string;
    customer_order_no: string;
    vendor_id: string;
    vendor_name: string;
    status: string;
    payment_status: string;
    ordered_date: string | null;
    expected_delivery_date: string | null;
    invoice_no: string | null;
    landing_cost: number;
    selling_total: number;
    gross_margin_pct: number;
    total_qty: number;
    sku_count: number;
}

export interface PaginatedPurchaseOrders {
    orders: PurchaseOrder[];
    page: number;
    limit: number;
    total: number;
    total_pages: number;
}

export interface PurchaseOrderFilters {
    status?: string;
    paymentStatus?: string;
    vendorId?: string;
    from?: string;
    to?: string;
    search?: string;
}

export const PO_STATUSES = [
    "PO Raised",
    "Waiting for Vendor",
    "Received",
    "Verified",
    "Pending Approval",
    "Approved",
    "Paid",
    "Rejected",
    "Cancelled",
    "Hold",
] as const;

export const PO_PAYMENT_STATUSES = ["Unpaid", "Payment Ready", "Paid"] as const;

export interface ChargeDetail {
    charge_id: number;
    charge_type_id: string;
    charge_name: string;
    rate_per_piece: number;
}

export interface LineItemDetail {
    line_item_id: number;
    sku_code: string | null;
    product_name: string;
    quantity: number;
    rate_per_unit: number;
    selling_price_per_unit: number;
    packaging_flat: number;
    charges: ChargeDetail[];
    computed_line_total: number;
}

export interface PurchaseOrderDetail extends PurchaseOrder {
  line_items: LineItemDetail[];
  computed_grand_total: number;
  stored_landing_cost: number;
  discrepancy: number;
}

export interface CreateChargeRequest {
  charge_type_id: string;
  rate_per_piece: number;
}

export interface CreateSKURequest {
  sku_code?: string;
  product_name: string;
  quantity: number;
  rate_per_unit: number;
  packaging_flat?: number;
  selling_price_per_unit?: number;
  charges?: CreateChargeRequest[];
}

export interface CreatePurchaseOrderRequest {
  customer_order_no: string;
  vendor_id: string;
  gst_pct?: number;
  ordered_date?: string;
  expected_delivery_date?: string;
  remarks?: string;
  skus: CreateSKURequest[];
}

export interface UpdateChargeRequest {
  charge_id?: number;
  charge_type_id?: string;
  rate_per_piece?: number;
}

export interface UpdateSKURequest {
  line_item_id?: number;
  sku_code?: string;
  product_name?: string;
  quantity?: number;
  rate_per_unit?: number;
  packaging_flat?: number;
  selling_price_per_unit?: number;
  charges?: UpdateChargeRequest[];
}

export interface UpdatePurchaseOrderRequest {
  po_number?: string;
  customer_order_no?: string;
  vendor_id?: string;
  status?: string;
  payment_status?: string;
  gst_pct?: number;
  ordered_date?: string;
  expected_delivery_date?: string;
  remarks?: string;
  skus?: UpdateSKURequest[];
}