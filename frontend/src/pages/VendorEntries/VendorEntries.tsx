import { useEffect, useState } from "react";
import { Plus, Search, Download, Eye, Pencil, ChevronLeft, ChevronRight } from "lucide-react";
import DashboardLayout from "../../layouts/DashboardLayout";
import StatusBadge from "../../components/StatusBadge";
import PurchaseOrderDetailModal from "../../components/PurchaseOrderDetailModal";
import { getPurchaseOrders, exportPurchaseOrders } from "../../api/purchaseOrderApi";
import { getVendors } from "../../api/vendorApi";
import { PO_STATUSES, PO_PAYMENT_STATUSES } from "../../types/purchaseOrder";
import type { PaginatedPurchaseOrders, PurchaseOrderFilters } from "../../types/purchaseOrder";
import type { Vendor } from "../../types/vendor";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const LIMIT = 10;

export default function VendorEntries() {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [status, setStatus] = useState("");
  const [vendorId, setVendorId] = useState("");
  const [paymentStatus, setPaymentStatus] = useState("");
  const [page, setPage] = useState(1);

  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [data, setData] = useState<PaginatedPurchaseOrders | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [viewingPO, setViewingPO] = useState<string | null>(null);

  useEffect(() => {
    getVendors().then(setVendors).catch(() => setVendors([]));
  }, []);

  useEffect(() => {
    const t = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [search]);

  useEffect(() => {
    setPage(1);
  }, [status, vendorId, paymentStatus]);

  useEffect(() => {
    const filters: PurchaseOrderFilters = {
      status: status || undefined,
      paymentStatus: paymentStatus || undefined,
      vendorId: vendorId || undefined,
      search: debouncedSearch || undefined,
    };
    setLoading(true);
    setError(null);
    getPurchaseOrders(filters, page, LIMIT)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Something went wrong"))
      .finally(() => setLoading(false));
  }, [status, vendorId, paymentStatus, debouncedSearch, page]);

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportPurchaseOrders({
        status: status || undefined,
        paymentStatus: paymentStatus || undefined,
        vendorId: vendorId || undefined,
        search: debouncedSearch || undefined,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to export");
    } finally {
      setExporting(false);
    }
  };

  const orders = data?.orders ?? [];

  return (
    <DashboardLayout breadcrumb="Vendor Entries">
      <div className="p-6">
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-2xl font-semibold">Vendor Entries</h1>
            <p className="text-gray-500 text-sm mt-1">
              Every procurement record. Order numbers are created internally and can be updated
              here.
            </p>
          </div>
          <button
            disabled
            title="Add Vendor Entry — coming soon"
            className="flex items-center gap-2 bg-black text-white text-sm px-4 py-2 rounded-lg opacity-50 cursor-not-allowed"
          >
            <Plus size={16} /> Add Vendor Entry
          </button>
        </div>

        <div className="border border-gray-200 rounded-xl overflow-hidden">
          <div className="flex flex-wrap items-center gap-3 p-4 border-b border-gray-200">
            <div className="relative flex-1 min-w-[200px]">
              <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search..."
                className="w-full pl-9 pr-3 py-2 text-sm border border-gray-200 rounded-lg"
              />
            </div>

            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="text-sm border border-gray-200 rounded-lg px-3 py-2"
            >
              <option value="">Status: All</option>
              {PO_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>

            <select
              value={vendorId}
              onChange={(e) => setVendorId(e.target.value)}
              className="text-sm border border-gray-200 rounded-lg px-3 py-2"
            >
              <option value="">Vendor: All</option>
              {vendors.map((v) => (
                <option key={v.vendor_id} value={v.vendor_id}>
                  {v.name}
                </option>
              ))}
            </select>

            <select
              value={paymentStatus}
              onChange={(e) => setPaymentStatus(e.target.value)}
              className="text-sm border border-gray-200 rounded-lg px-3 py-2"
            >
              <option value="">Payment: All</option>
              {PO_PAYMENT_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>

            <button
              onClick={handleExport}
              disabled={exporting}
              className="flex items-center gap-1.5 text-sm border border-gray-200 rounded-lg px-3 py-2 hover:bg-gray-50 disabled:opacity-50"
            >
              <Download size={14} /> {exporting ? "Exporting…" : "CSV"}
            </button>

            {data && (
              <span className="text-sm text-gray-400 ml-auto whitespace-nowrap">
                {data.total} record{data.total === 1 ? "" : "s"}
              </span>
            )}
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-[11px] tracking-wide text-gray-400">
                  <th className="px-4 py-3 font-medium">ORDER NO</th>
                  <th className="px-4 py-3 font-medium">INVOICE NO</th>
                  <th className="px-4 py-3 font-medium">VENDOR</th>
                  <th className="px-4 py-3 font-medium text-right">SKUS</th>
                  <th className="px-4 py-3 font-medium text-right">TOTAL QTY</th>
                  <th className="px-4 py-3 font-medium text-right">LANDING COST</th>
                  <th className="px-4 py-3 font-medium text-right">SELLING</th>
                  <th className="px-4 py-3 font-medium text-right">MARGIN</th>
                  <th className="px-4 py-3 font-medium">STATUS</th>
                  <th className="px-4 py-3 font-medium text-right">ACTIONS</th>
                </tr>
              </thead>
              <tbody>
                {loading && (
                  <tr>
                    <td colSpan={10} className="px-4 py-8 text-center text-gray-400">
                      Loading…
                    </td>
                  </tr>
                )}
                {!loading && error && (
                  <tr>
                    <td colSpan={10} className="px-4 py-8 text-center text-red-600">
                      {error}
                    </td>
                  </tr>
                )}
                {!loading && !error && orders.length === 0 && (
                  <tr>
                    <td colSpan={10} className="px-4 py-8 text-center text-gray-400">
                      No purchase orders match these filters.
                    </td>
                  </tr>
                )}
                {!loading &&
                  !error &&
                  orders.map((po) => (
                    <tr key={po.po_number} className="border-t border-gray-100">
                      <td className="px-4 py-3">
                        <p className="font-medium">{po.po_number}</p>
                        <p className="text-xs text-gray-400">{po.customer_order_no}</p>
                      </td>
                      <td className="px-4 py-3 text-gray-600">{po.invoice_no ?? "—"}</td>
                      <td className="px-4 py-3">{po.vendor_name}</td>
                      <td className="px-4 py-3 text-right">{po.sku_count}</td>
                      <td className="px-4 py-3 text-right">{po.total_qty}</td>
                      <td className="px-4 py-3 text-right">{currency.format(po.landing_cost)}</td>
                      <td className="px-4 py-3 text-right">{currency.format(po.selling_total)}</td>
                      <td
                        className={`px-4 py-3 text-right font-medium ${
                          po.gross_margin_pct < 0 ? "text-red-600" : "text-gray-900"
                        }`}
                      >
                        {po.gross_margin_pct.toFixed(1)}%
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={po.payment_status === "Paid" ? "Paid" : po.status} />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => setViewingPO(po.po_number)}
                            className="text-gray-400 hover:text-gray-700"
                            title="View"
                          >
                            <Eye size={15} />
                          </button>
                          <button
                            disabled
                            title="Edit — coming soon"
                            className="text-gray-200 cursor-not-allowed"
                          >
                            <Pencil size={15} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>

          {data && data.total_pages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200 text-sm text-gray-500">
              <span>
                Page {data.page} of {data.total_pages}
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="flex items-center gap-1 border border-gray-200 rounded-lg px-3 py-1.5 disabled:opacity-40"
                >
                  <ChevronLeft size={14} /> Prev
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(data.total_pages, p + 1))}
                  disabled={page >= data.total_pages}
                  className="flex items-center gap-1 border border-gray-200 rounded-lg px-3 py-1.5 disabled:opacity-40"
                >
                  Next <ChevronRight size={14} />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {viewingPO && (
        <PurchaseOrderDetailModal poNumber={viewingPO} onClose={() => setViewingPO(null)} />
      )}
    </DashboardLayout>
  );
}