import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { getPurchaseOrderDetail } from "../api/purchaseOrderApi";
import type { PurchaseOrderDetail } from "../types/purchaseOrder";
import StatusBadge from "./StatusBadge";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

function formatDate(value: string | null) {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric" });
}

export default function PurchaseOrderDetailModal({
  poNumber,
  onClose,
}: {
  poNumber: string;
  onClose: () => void;
}) {
  const [detail, setDetail] = useState<PurchaseOrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    getPurchaseOrderDetail(poNumber)
      .then(setDetail)
      .catch((err) => setError(err instanceof Error ? err.message : "Something went wrong"))
      .finally(() => setLoading(false));
  }, [poNumber]);

  return (
    <div className="fixed inset-0 bg-black/30 z-50 flex justify-end">
      <div className="bg-white h-full w-full max-w-xl overflow-y-auto p-6">
        <div className="flex justify-between items-start mb-1">
          <div className="flex items-center gap-3">
            <div>
              <h2 className="text-lg font-semibold">{poNumber}</h2>
              <p className="text-sm text-gray-500">
                {detail ? `${detail.customer_order_no} · ${detail.vendor_name}` : "Purchase order detail"}
              </p>
            </div>
            {detail && (
              <StatusBadge status={detail.payment_status === "Paid" ? "Paid" : detail.status} />
            )}
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X size={18} />
          </button>
        </div>

        {loading && <p className="text-gray-400 text-sm mt-6">Loading…</p>}
        {error && <p className="text-red-600 text-sm mt-6">{error}</p>}

        {detail && !loading && !error && (
          <div className="mt-4 space-y-6">
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <p className="text-gray-400 text-xs">SKUS / LINES</p>
                <p className="mt-1.5 font-medium">
                  {detail.sku_count} · {detail.total_qty} units total
                </p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">CUSTOMER ORDER</p>
                <p className="mt-1.5 font-medium">{detail.customer_order_no}</p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">ORDERED</p>
                <p className="mt-1.5 font-medium">{formatDate(detail.ordered_date)}</p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">RECEIVED</p>
                <p className="mt-1.5 font-medium">{formatDate(detail.received_date)}</p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">INVOICE</p>
                <p className="mt-1.5 font-medium">{detail.invoice_no ?? "—"}</p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">DUE DATE</p>
                <p className="mt-1.5 font-medium">{formatDate(detail.due_date)}</p>
              </div>
            </div>

            <div>
              <p className="text-xs text-gray-400 mb-3">SKUS ({(detail.line_items ?? []).length})</p>
              <div className="space-y-3">
                {(detail.line_items ?? []).map((li) => {
                  const sellingTotal = li.selling_price_per_unit * li.quantity;
                  const marginPct =
                    sellingTotal > 0
                      ? ((sellingTotal - li.computed_line_total) / sellingTotal) * 100
                      : 0;
                  return (
                    <div key={li.line_item_id} className="bg-gray-50 rounded-lg p-4 text-sm">
                      <div className="flex justify-between items-baseline mb-2">
                        <p className="font-semibold">{li.product_name}</p>
                        <p className="text-xs text-gray-400">{li.sku_code ?? "—"}</p>
                      </div>

                      <div className="flex justify-between text-gray-700">
                        <span>
                          {li.quantity} × {currency.format(li.rate_per_unit)}
                        </span>
                        <span>{currency.format(li.rate_per_unit * li.quantity)}</span>
                      </div>

                      {li.charges.map((c) => (
                        <div key={c.charge_id} className="flex justify-between text-xs text-gray-500 pl-3 mt-1">
                          <span>
                            {c.charge_name} @ {currency.format(c.rate_per_piece)}/pc × {li.quantity}
                          </span>
                          <span>{currency.format(c.rate_per_piece * li.quantity)}</span>
                        </div>
                      ))}

                      {li.packaging_flat > 0 && (
                        <div className="flex justify-between text-xs text-gray-500 pl-3 mt-1">
                          <span>Packaging (flat)</span>
                          <span>{currency.format(li.packaging_flat)}</span>
                        </div>
                      )}

                      <div className="flex justify-between font-medium border-t border-gray-200 mt-2 pt-2">
                        <span>Line landing</span>
                        <span>{currency.format(li.computed_line_total)}</span>
                      </div>

                      <div className="flex justify-between text-gray-600 mt-1">
                        <span>
                          Selling {currency.format(li.selling_price_per_unit)}/unit · margin{" "}
                          {marginPct.toFixed(1)}%
                        </span>
                        <span className="font-semibold text-gray-900">
                          {currency.format(sellingTotal)}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4 border-t border-gray-100 pt-4 text-sm">
              <div>
                <p className="text-gray-400 text-xs">LANDING COST</p>
                <p className="font-semibold mt-1">{currency.format(detail.landing_cost)}</p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">SELLING TOTAL</p>
                <p className="font-semibold mt-1">{currency.format(detail.selling_total)}</p>
              </div>
              <div>
                <p className="text-gray-400 text-xs">GROSS MARGIN</p>
                <p className="font-semibold mt-1">{detail.gross_margin_pct.toFixed(1)}%</p>
              </div>
            </div>

            {Math.abs(detail.discrepancy) > 0.01 && (
              <p className="text-xs text-amber-600 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2">
                Stored landing cost ({currency.format(detail.stored_landing_cost)}) differs from the
                recomputed total ({currency.format(detail.computed_grand_total)}) by{" "}
                {currency.format(detail.discrepancy)}.
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}