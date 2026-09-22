import { useEffect, useMemo, useState } from "react";
import { X, Plus, Trash2 } from "lucide-react";
import { getPurchaseOrderDetail, updatePurchaseOrder } from "../api/purchaseOrderApi";
import { getVendors } from "../api/vendorApi";
import { getChargeTypes } from "../api/chargeApi";
import { PO_STATUSES, PO_PAYMENT_STATUSES } from "../types/purchaseOrder";
import type { UpdatePurchaseOrderRequest, UpdateSKURequest } from "../types/purchaseOrder";
import type { Vendor } from "../types/vendor";
import type { ChargeType } from "../types/chargeType";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

interface Props {
  poNumber: string;
  onClose: () => void;
  onUpdated: () => void;
}

interface ChargeCell {
  charge_id?: number; // present = update existing charge row, absent = insert a new one
  rate: number;
}

interface EditSKURow {
  line_item_id?: number; // present = existing line item (update), absent = new line item (insert)
  sku_code: string;
  product_name: string;
  quantity: number;
  rate_per_unit: number;
  selling_price_per_unit: number;
  packaging_flat: number;
  chargeRates: Record<string, ChargeCell>; // keyed by charge_type_id
}

const emptySku = (): EditSKURow => ({
  sku_code: "",
  product_name: "",
  quantity: 0,
  rate_per_unit: 0,
  selling_price_per_unit: 0,
  packaging_flat: 0,
  chargeRates: {},
});

function parseNumberInput(raw: string): number {
  const cleaned = raw.replace(/^0+(?=\d)/, "");
  return cleaned === "" ? 0 : Number(cleaned);
}

function skuBase(s: EditSKURow) {
  return s.quantity * s.rate_per_unit;
}
function skuChargesTotal(s: EditSKURow) {
  return Object.values(s.chargeRates).reduce((sum, c) => sum + (c.rate || 0) * s.quantity, 0);
}
function skuLandingTotal(s: EditSKURow) {
  return skuBase(s) + skuChargesTotal(s) + (s.packaging_flat || 0);
}
function skuSellingTotal(s: EditSKURow) {
  return s.selling_price_per_unit * s.quantity;
}
function skuMarginPct(s: EditSKURow) {
  const sel = skuSellingTotal(s);
  return sel > 0 ? ((sel - skuLandingTotal(s)) / sel) * 100 : 0;
}

export default function EditPurchaseOrderModal({ poNumber, onClose, onUpdated }: Props) {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [chargeTypes, setChargeTypes] = useState<ChargeType[]>([]);

  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [customerOrderNo, setCustomerOrderNo] = useState("");
  const [vendorId, setVendorId] = useState("");
  const [status, setStatus] = useState("");
  const [paymentStatus, setPaymentStatus] = useState("");
  const [gstPct, setGstPct] = useState(18);
  const [orderedDate, setOrderedDate] = useState("");
  const [expectedDeliveryDate, setExpectedDeliveryDate] = useState("");
  const [remarks, setRemarks] = useState("");
  const [skus, setSkus] = useState<EditSKURow[]>([]);

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getVendors().then(setVendors).catch(() => setVendors([]));
    getChargeTypes().then(setChargeTypes).catch(() => setChargeTypes([]));

    setLoading(true);
    setLoadError(null);
    getPurchaseOrderDetail(poNumber)
      .then((detail) => {
        setCustomerOrderNo(detail.customer_order_no);
        setVendorId(detail.vendor_id);
        setStatus(detail.status);
        setPaymentStatus(detail.payment_status);
        setOrderedDate(detail.ordered_date ?? "");
        setExpectedDeliveryDate(detail.expected_delivery_date ?? "");

        setSkus(
          detail.line_items.map((li) => ({
            line_item_id: li.line_item_id,
            sku_code: li.sku_code ?? "",
            product_name: li.product_name,
            quantity: li.quantity,
            rate_per_unit: li.rate_per_unit,
            selling_price_per_unit: li.selling_price_per_unit,
            packaging_flat: li.packaging_flat,
            chargeRates: Object.fromEntries(
              li.charges.map((c) => [
                c.charge_type_id,
                { charge_id: c.charge_id, rate: c.rate_per_piece },
              ])
            ),
          }))
        );
      })
      .catch((err) => setLoadError(err instanceof Error ? err.message : "Failed to load purchase order"))
      .finally(() => setLoading(false));
  }, [poNumber]);

  const updateSku = (i: number, patch: Partial<EditSKURow>) =>
    setSkus((prev) => prev.map((s, idx) => (idx === i ? { ...s, ...patch } : s)));

  const updateChargeRate = (skuIndex: number, chargeTypeId: string, rate: number) =>
    setSkus((prev) =>
      prev.map((s, idx) => {
        if (idx !== skuIndex) return s;
        const existing = s.chargeRates[chargeTypeId];
        return {
          ...s,
          chargeRates: {
            ...s.chargeRates,
            [chargeTypeId]: { charge_id: existing?.charge_id, rate },
          },
        };
      })
    );

  const addSku = () => setSkus((prev) => [...prev, emptySku()]);

  // Only newly-added rows (no line_item_id yet) can be removed from the form —
  // the PATCH endpoint has no delete support for line items already saved to the DB.
  const removeSku = (i: number) =>
    setSkus((prev) => prev.filter((s, idx) => !(idx === i && s.line_item_id === undefined)));

  const totals = useMemo(() => {
    const totalQty = skus.reduce((s, sku) => s + (sku.quantity || 0), 0);
    const base = skus.reduce((s, sku) => s + skuBase(sku), 0);
    const packaging = skus.reduce((s, sku) => s + (sku.packaging_flat || 0), 0);
    const charges = skus.reduce((s, sku) => s + skuChargesTotal(sku), 0);
    const gross = base + packaging + charges;
    const gst = base * (gstPct / 100); // matches backend: GST applies to base only
    const landingCost = gross + gst;
    const sellingTotal = skus.reduce((s, sku) => s + skuSellingTotal(sku), 0);
    const grossProfit = sellingTotal - landingCost;
    const marginPct = sellingTotal > 0 ? (grossProfit / sellingTotal) * 100 : 0;
    const landingPerUnit = totalQty > 0 ? landingCost / totalQty : 0;
    return { totalQty, base, packaging, charges, gross, gst, landingCost, sellingTotal, grossProfit, marginPct, landingPerUnit };
  }, [skus, gstPct]);

  const validate = (): string | null => {
    if (!customerOrderNo.trim()) return "Customer order no. is required";
    if (!vendorId) return "Please select a vendor";
    if (skus.length === 0) return "Add at least one SKU";
    for (const s of skus) {
      if (!s.product_name.trim()) return "Every SKU needs a product name";
      if (!s.quantity || s.quantity <= 0) return "Quantity must be greater than 0";
      if (s.rate_per_unit < 0) return "Rate per unit cannot be negative";
    }
    return null;
  };

  const handleSubmit = async () => {
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const payloadSkus: UpdateSKURequest[] = skus.map((s) => ({
        line_item_id: s.line_item_id,
        sku_code: s.sku_code || undefined,
        product_name: s.product_name,
        quantity: s.quantity,
        rate_per_unit: s.rate_per_unit,
        packaging_flat: s.packaging_flat,
        selling_price_per_unit: s.selling_price_per_unit,
        charges: Object.entries(s.chargeRates)
          .filter(([, c]) => c.rate > 0)
          .map(([charge_type_id, c]) => ({
            charge_id: c.charge_id,
            charge_type_id,
            rate_per_piece: c.rate,
          })),
      }));

      const payload: UpdatePurchaseOrderRequest = {
        customer_order_no: customerOrderNo,
        vendor_id: vendorId,
        status,
        payment_status: paymentStatus,
        gst_pct: gstPct,
        ordered_date: orderedDate || undefined,
        expected_delivery_date: expectedDeliveryDate || undefined,
        remarks: remarks || undefined,
        skus: payloadSkus,
      };

      await updatePurchaseOrder(poNumber, payload);
      onUpdated();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update purchase order");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/30 z-50 flex items-start justify-center overflow-y-auto py-10">
      <div className="bg-white w-full max-w-4xl rounded-2xl p-6">
        <div className="flex justify-between items-start mb-1">
          <div>
            <h2 className="text-lg font-semibold">Edit Purchase Order · {poNumber}</h2>
            <p className="text-sm text-gray-500">Update order details, status, or line items.</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X size={18} />
          </button>
        </div>

        {loading && <p className="text-sm text-gray-400 mt-6">Loading purchase order…</p>}

        {!loading && loadError && (
          <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mt-4">
            {loadError}
          </p>
        )}

        {!loading && !loadError && (
          <>
            {error && (
              <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mt-4">
                {error}
              </p>
            )}

            <div className="flex gap-6 mt-6 items-start">
              {/* LEFT: form */}
              <div className="flex-1 min-w-0">
                <p className="text-xs font-semibold text-gray-400 mb-3">ORDER</p>

                <div className="grid grid-cols-3 gap-3 text-sm mb-3">
                  <div>
                    <label className="text-xs text-gray-500">Vendor *</label>
                    <select
                      value={vendorId}
                      onChange={(e) => setVendorId(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    >
                      <option value="">Select vendor</option>
                      {vendors.map((v) => (
                        <option key={v.vendor_id} value={v.vendor_id}>
                          {v.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="text-xs text-gray-500">Customer Order No *</label>
                    <input
                      value={customerOrderNo}
                      onChange={(e) => setCustomerOrderNo(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-gray-500">PO Number</label>
                    <input
                      value={poNumber}
                      disabled
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2 bg-gray-50 text-gray-400"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 text-sm mb-3">
                  <div>
                    <label className="text-xs text-gray-500">Status</label>
                    <select
                      value={status}
                      onChange={(e) => setStatus(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    >
                      {PO_STATUSES.map((s) => (
                        <option key={s} value={s}>
                          {s}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="text-xs text-gray-500">Payment Status</label>
                    <select
                      value={paymentStatus}
                      onChange={(e) => setPaymentStatus(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    >
                      {PO_PAYMENT_STATUSES.map((s) => (
                        <option key={s} value={s}>
                          {s}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-3 text-sm mb-6">
                  <div>
                    <label className="text-xs text-gray-500">Ordered Date</label>
                    <input
                      type="date"
                      value={orderedDate}
                      onChange={(e) => setOrderedDate(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-gray-500">Expected Delivery Date</label>
                    <input
                      type="date"
                      value={expectedDeliveryDate}
                      onChange={(e) => setExpectedDeliveryDate(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-gray-500">GST %</label>
                    <select
                      value={gstPct}
                      onChange={(e) => setGstPct(Number(e.target.value))}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    >
                      {[0, 5, 12, 18, 28].map((g) => (
                        <option key={g} value={g}>
                          {g}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="col-span-3">
                    <label className="text-xs text-gray-500">Remarks</label>
                    <input
                      value={remarks}
                      onChange={(e) => setRemarks(e.target.value)}
                      className="mt-1 w-full border border-gray-200 rounded-lg px-3 py-2"
                    />
                  </div>
                </div>

                <div className="flex justify-between items-center mb-3">
                  <p className="text-xs font-semibold text-gray-400">SKUS / LINE ITEMS</p>
                  <button
                    onClick={addSku}
                    className="flex items-center gap-1 text-xs border border-gray-200 rounded-lg px-2.5 py-1.5 hover:bg-gray-50"
                  >
                    <Plus size={13} /> Add SKU
                  </button>
                </div>

                <div className="space-y-4">
                  {skus.map((sku, i) => (
                    <div key={sku.line_item_id ?? `new-${i}`} className="bg-gray-50 rounded-lg p-4">
                      <div className="flex justify-between items-center mb-3">
                        <p className="font-semibold text-sm">
                          {sku.line_item_id ? `SKU #${sku.line_item_id}` : "New SKU"}
                        </p>
                        <button
                          onClick={() => removeSku(i)}
                          disabled={sku.line_item_id !== undefined}
                          title={
                            sku.line_item_id !== undefined
                              ? "Existing line items can't be deleted yet"
                              : "Remove this new SKU"
                          }
                          className="text-gray-300 hover:text-red-500 disabled:opacity-30"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>

                      <div className="grid grid-cols-2 gap-3 text-sm mb-3">
                        <div>
                          <label className="text-[10px] text-gray-400">SKU CODE</label>
                          <input
                            value={sku.sku_code}
                            onChange={(e) => updateSku(i, { sku_code: e.target.value })}
                            placeholder="WC-XXXX"
                            className="mt-1 w-full border border-gray-200 rounded-lg px-2 py-1.5"
                          />
                        </div>
                        <div>
                          <label className="text-[10px] text-gray-400">PRODUCT NAME *</label>
                          <input
                            value={sku.product_name}
                            onChange={(e) => updateSku(i, { product_name: e.target.value })}
                            placeholder="e.g. Royal Scroll Wedding Card"
                            className="mt-1 w-full border border-gray-200 rounded-lg px-2 py-1.5"
                          />
                        </div>
                      </div>

                      <div className="grid grid-cols-3 gap-3 text-sm mb-3">
                        <div>
                          <label className="text-[10px] text-gray-400">QUANTITY *</label>
                          <input
                            type="number"
                            value={sku.quantity === 0 ? "" : sku.quantity}
                            onChange={(e) => updateSku(i, { quantity: parseNumberInput(e.target.value) })}
                            className="mt-1 w-full border border-gray-200 rounded-lg px-2 py-1.5"
                          />
                        </div>
                        <div>
                          <label className="text-[10px] text-gray-400">RATE / UNIT *</label>
                          <input
                            type="number"
                            value={sku.rate_per_unit === 0 ? "" : sku.rate_per_unit}
                            onChange={(e) => updateSku(i, { rate_per_unit: parseNumberInput(e.target.value) })}
                            className="mt-1 w-full border border-gray-200 rounded-lg px-2 py-1.5"
                          />
                        </div>
                        <div>
                          <label className="text-[10px] text-gray-400">SELLING / UNIT *</label>
                          <input
                            type="number"
                            value={sku.selling_price_per_unit === 0 ? "" : sku.selling_price_per_unit}
                            onChange={(e) =>
                              updateSku(i, { selling_price_per_unit: parseNumberInput(e.target.value) })
                            }
                            className="mt-1 w-full border border-gray-200 rounded-lg px-2 py-1.5"
                          />
                        </div>
                      </div>

                      <div className="mb-3">
                        <label className="text-[10px] text-gray-400">PACKAGING (FLAT)</label>
                        <input
                          type="number"
                          value={sku.packaging_flat === 0 ? "" : sku.packaging_flat}
                          onChange={(e) => updateSku(i, { packaging_flat: parseNumberInput(e.target.value) })}
                          className="mt-1 w-40 border border-gray-200 rounded-lg px-2 py-1.5"
                        />
                      </div>

                      {chargeTypes.length > 0 && (
                        <>
                          <p className="text-[10px] tracking-wide text-gray-400 border-t border-gray-200 pt-3 mb-2">
                            ADDITIONAL CHARGES · COST PER PIECE · MULTIPLIED BY QUANTITY
                          </p>
                          <div className="grid grid-cols-4 gap-2">
                            {chargeTypes.map((ct) => {
                              const cell = sku.chargeRates[ct.charge_type_id];
                              const rate = cell?.rate ?? 0;
                              return (
                                <div key={ct.charge_type_id}>
                                  <label className="text-[10px] text-gray-400">{ct.name}</label>
                                  <div className="flex items-center gap-1 mt-1">
                                    <input
                                      type="number"
                                      value={rate === 0 ? "" : rate}
                                      onChange={(e) =>
                                        updateChargeRate(i, ct.charge_type_id, parseNumberInput(e.target.value))
                                      }
                                      className="w-full border border-gray-200 rounded-lg px-2 py-1.5 text-sm"
                                    />
                                    <span className="text-xs text-gray-400 whitespace-nowrap">
                                      = ₹{(rate * sku.quantity).toFixed(0)}
                                    </span>
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        </>
                      )}

                      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 border-t border-gray-200 mt-3 pt-2">
                        <span>Base ₹{skuBase(sku).toFixed(0)}</span>
                        <span>Charges ₹{skuChargesTotal(sku).toFixed(0)}</span>
                        <span>Packaging ₹{(sku.packaging_flat || 0).toFixed(0)}</span>
                        <span>Line landing ₹{skuLandingTotal(sku).toFixed(0)}</span>
                        <span>Line selling ₹{skuSellingTotal(sku).toFixed(0)}</span>
                      </div>
                      <p className="text-xs font-medium mt-1">Margin {skuMarginPct(sku).toFixed(1)}%</p>
                    </div>
                  ))}
                </div>
              </div>

              {/* RIGHT: live cost summary */}
              <div className="w-64 shrink-0 border border-gray-200 rounded-xl p-4 sticky top-0">
                <p className="font-semibold mb-3">Live Cost Summary</p>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">SKUs</span>
                    <span>{skus.length}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Total Qty</span>
                    <span>{totals.totalQty}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Base</span>
                    <span>{currency.format(totals.base)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Packaging</span>
                    <span>{currency.format(totals.packaging)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Additional charges</span>
                    <span>{currency.format(totals.charges)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Gross</span>
                    <span>{currency.format(totals.gross)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">GST</span>
                    <span>{currency.format(totals.gst)}</span>
                  </div>
                  <div className="flex justify-between font-semibold border-t border-gray-200 pt-2">
                    <span>Landing Cost</span>
                    <span>{currency.format(totals.landingCost)}</span>
                  </div>
                </div>

                <div className="bg-gray-50 rounded-lg text-center py-4 mt-4">
                  <p className="text-2xl font-bold">{totals.marginPct.toFixed(1)}%</p>
                  <p className="text-xs text-gray-400 mt-1">GROSS MARGIN</p>
                </div>

                <div className="text-sm space-y-1 mt-3">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Gross Profit</span>
                    <span>{currency.format(totals.grossProfit)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Landing / unit</span>
                    <span>{currency.format(totals.landingPerUnit)}</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex justify-end gap-2 border-t border-gray-100 mt-6 pt-4">
              <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg border border-gray-200">
                Cancel
              </button>
              <button
                onClick={handleSubmit}
                disabled={saving}
                className="text-sm px-4 py-2 rounded-lg bg-black text-white disabled:opacity-50"
              >
                {saving ? "Saving…" : "Save Changes"}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}