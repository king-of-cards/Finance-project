import { useEffect, useMemo, useState } from "react";
import { X, Check } from "lucide-react";
import {
  getPurchaseOrderDetail,
  approvePurchaseOrder,
  markPurchaseOrderPaid,
} from "../api/purchaseOrderApi";
import type { PurchaseOrderDetail } from "../types/purchaseOrder";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

interface Props {
  poNumber: string;
  onClose: () => void;
  onActionComplete: () => void;
}

const STEPS = ["Finance Entry", "Verification", "Admin Review", "Approval", "Payment Ready", "Paid"];

// Maps a PO's current status/payment_status to how far along the 6-step workflow it is,
// purely for the stepper display at the top of the drawer.
function currentStepIndex(status: string, paymentStatus: string): number {
  if (paymentStatus === "Paid") return 5;
  if (status === "Approved") return 4; // sitting at "Payment Ready"
  if (status === "Verified" || status === "Pending Approval") return 3; // needs Approval
  if (status === "Received") return 2; // needs Verification
  return 1; // PO Raised / Waiting for Vendor — still at Finance Entry
}

export default function PaymentApprovalDrawer({ poNumber, onClose, onActionComplete }: Props) {
  const [detail, setDetail] = useState<PurchaseOrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [verified, setVerified] = useState<Set<number>>(new Set());
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setLoadError(null);
    getPurchaseOrderDetail(poNumber)
      .then((d) => {
        setDetail(d);
        // If the PO is already past the Approval step, all lines are implicitly verified.
        if (d.status === "Approved" || d.payment_status === "Paid") {
          setVerified(new Set(d.line_items.map((li) => li.line_item_id)));
        }
      })
      .catch((err) => setLoadError(err instanceof Error ? err.message : "Failed to load purchase order"))
      .finally(() => setLoading(false));
  }, [poNumber]);

  const toggleVerified = (lineItemId: number) =>
    setVerified((prev) => {
      const next = new Set(prev);
      if (next.has(lineItemId)) next.delete(lineItemId);
      else next.add(lineItemId);
      return next;
    });

  // Cost breakdown, derived from the fields the API actually returns:
  // - base = sum(qty * rate_per_unit) across line items
  // - packaging & charges = computed_grand_total - base  (computed_grand_total already
  //   includes qty*packaging_flat + qty*charges, see db.GetPurchaseOrderDetail)
  // - GST = stored_landing_cost - computed_grand_total  (landing_cost is the only value
  //   that has GST baked in; computed_grand_total deliberately excludes it)
  // - selling total = sum(qty * selling_price_per_unit), since the API's own
  //   `selling_total` field on the list/detail response isn't populated
  const breakdown = useMemo(() => {
    if (!detail) return null;
    const base = detail.line_items.reduce((sum, li) => sum + li.quantity * li.rate_per_unit, 0);
    const packagingAndCharges = detail.computed_grand_total - base;
    const gst = detail.stored_landing_cost - detail.computed_grand_total;
    const totalPayable = detail.stored_landing_cost;
    const sellingTotal = detail.line_items.reduce(
      (sum, li) => sum + li.quantity * li.selling_price_per_unit,
      0
    );
    const grossProfit = sellingTotal - totalPayable;
    const marginPct = sellingTotal > 0 ? (grossProfit / sellingTotal) * 100 : 0;
    const landingPerUnit = detail.total_qty > 0 ? totalPayable / detail.total_qty : 0;
    return { base, packagingAndCharges, gst, totalPayable, sellingTotal, grossProfit, marginPct, landingPerUnit };
  }, [detail]);

  const allVerified = detail ? detail.line_items.every((li) => verified.has(li.line_item_id)) : false;
  const isPaid = detail?.payment_status === "Paid";
  const isApproved = detail?.status === "Approved";

  const handleApprove = async () => {
    if (!detail || !allVerified) return;
    setSubmitting(true);
    setActionError(null);
    try {
      await approvePurchaseOrder(poNumber, Array.from(verified), comment || undefined);
      onActionComplete();
      onClose();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to approve purchase order");
    } finally {
      setSubmitting(false);
    }
  };

  const handleMarkPaid = async () => {
    setSubmitting(true);
    setActionError(null);
    try {
      await markPurchaseOrderPaid(poNumber, comment || undefined);
      onActionComplete();
      onClose();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to mark purchase order as paid");
    } finally {
      setSubmitting(false);
    }
  };

  const stepIdx = detail ? currentStepIndex(detail.status, detail.payment_status) : 0;

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative bg-white h-full w-full max-w-xl overflow-y-auto p-6">
        <div className="flex justify-between items-start mb-4">
          <div>
            <h2 className="text-lg font-semibold">
              {isPaid ? "Payment Details" : isApproved ? "Mark as Paid" : "Approve Payment"}
            </h2>
            <p className="text-sm text-gray-500">
              {poNumber} · {detail?.vendor_name ?? "…"}
            </p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X size={18} />
          </button>
        </div>

        {loading && <p className="text-sm text-gray-400">Loading…</p>}
        {!loading && loadError && (
          <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
            {loadError}
          </p>
        )}

        {!loading && detail && breakdown && (
          <>
            {/* Workflow stepper */}
            <div className="flex items-center mb-6 mt-2">
              {STEPS.map((step, i) => {
                const done = i < stepIdx;
                const active = i === stepIdx;
                return (
                  <div key={step} className="flex items-center flex-1 last:flex-none">
                    <div className="flex flex-col items-center gap-1">
                      <div
                        className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-medium ${
                          done
                            ? "bg-black text-white"
                            : active
                            ? "border-2 border-black text-black"
                            : "border border-gray-300 text-gray-400"
                        }`}
                      >
                        {done ? <Check size={13} /> : i + 1}
                      </div>
                      <span className="text-[10px] text-gray-400 whitespace-nowrap">{step}</span>
                    </div>
                    {i < STEPS.length - 1 && (
                      <div className={`flex-1 h-px mx-1 mb-4 ${done ? "bg-black" : "bg-gray-200"}`} />
                    )}
                  </div>
                );
              })}
            </div>

            {actionError && (
              <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mb-4">
                {actionError}
              </p>
            )}

            {/* Invoice / cost breakdown */}
            <div className="border border-gray-200 rounded-lg p-4 mb-6 text-sm">
              <div className="flex justify-between mb-3">
                <span className="font-medium">Invoice {detail.invoice_no ?? "—"}</span>
                <span className="text-gray-400">{detail.ordered_date ?? ""}</span>
              </div>
              <div className="flex justify-between text-gray-500 mb-2">
                <span>
                  {detail.line_items.length} SKU line(s) · {detail.total_qty} units
                </span>
                <span>{currency.format(breakdown.base)}</span>
              </div>
              <div className="flex justify-between text-gray-500 mb-2">
                <span>Packaging & additional charges</span>
                <span>{currency.format(breakdown.packagingAndCharges)}</span>
              </div>
              <div className="flex justify-between text-gray-500 mb-3">
                <span>GST</span>
                <span>{currency.format(breakdown.gst)}</span>
              </div>
              <div className="flex justify-between font-semibold border-t border-gray-200 pt-3">
                <span>Total Payable</span>
                <span>{currency.format(breakdown.totalPayable)}</span>
              </div>
            </div>

            {/* Per-SKU verification checklist */}
            <div className="mb-6">
              <p className="text-xs font-semibold text-gray-400 mb-1">
                RECEIVED DETAILS — VERIFY EACH SKU
              </p>
              <p className="text-xs text-gray-400 mb-3">
                Tick every line to confirm goods were received as invoiced. Approval unlocks only
                when all are checked.
              </p>
              <div className="space-y-2">
                {detail.line_items.map((li) => (
                  <label
                    key={li.line_item_id}
                    className="flex items-start gap-3 border border-gray-200 rounded-lg p-3 cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      checked={verified.has(li.line_item_id)}
                      onChange={() => toggleVerified(li.line_item_id)}
                      disabled={isApproved || isPaid}
                      className="mt-1"
                    />
                    <div className="flex-1 text-sm">
                      <p className="font-medium">{li.product_name}</p>
                      <p className="text-xs text-gray-400">{li.sku_code ?? "—"}</p>
                      <p className="text-xs text-gray-500 mt-0.5">
                        {li.quantity} units · {currency.format(li.computed_line_total)}
                      </p>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            {/* Margin analysis */}
            <div className="mb-6">
              <p className="text-xs font-semibold text-gray-400 mb-3">MARGIN ANALYSIS</p>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="text-xs text-gray-400">LANDING / UNIT</p>
                  <p className="font-medium">{currency.format(breakdown.landingPerUnit)}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-400">SELLING PRICE</p>
                  <p className="font-medium">{currency.format(breakdown.sellingTotal)}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-400">GROSS PROFIT</p>
                  <p className="font-medium">{currency.format(breakdown.grossProfit)}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-400">GROSS MARGIN</p>
                  <p className="font-medium">{breakdown.marginPct.toFixed(1)}%</p>
                </div>
              </div>
            </div>

            {!isPaid && (
              <div className="mb-6">
                <label className="text-xs text-gray-400">COMMENT (OPTIONAL)</label>
                <input
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2 text-sm"
                />
              </div>
            )}

            {/* Action */}
            {isPaid ? (
              <div className="text-sm text-gray-500 bg-gray-50 rounded-lg p-3">
                This purchase order has already been paid.
              </div>
            ) : isApproved ? (
              <div className="flex justify-end gap-2 border-t border-gray-100 pt-4">
                <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg border border-gray-200">
                  Cancel
                </button>
                <button
                  onClick={handleMarkPaid}
                  disabled={submitting}
                  className="text-sm px-4 py-2 rounded-lg bg-black text-white disabled:opacity-50"
                >
                  {submitting ? "Marking as Paid…" : "Mark as Paid"}
                </button>
              </div>
            ) : (
              <div className="flex justify-end gap-2 border-t border-gray-100 pt-4">
                <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg border border-gray-200">
                  Cancel
                </button>
                <button
                  onClick={handleApprove}
                  disabled={submitting || !allVerified}
                  title={!allVerified ? "Tick every SKU line before approving" : undefined}
                  className="text-sm px-4 py-2 rounded-lg bg-black text-white disabled:opacity-50"
                >
                  {submitting ? "Approving…" : "Approve Payment"}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}