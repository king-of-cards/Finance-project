import { useEffect, useState } from "react";
import { Search, Download, Check } from "lucide-react";
import DashboardLayout from "../../layouts/DashboardLayout";
import StatusBadge from "../../components/StatusBadge";
import PaymentApprovalDrawer from "../../components/PaymentApprovalDrawer";
import { getPurchaseOrders, exportPurchaseOrders } from "../../api/purchaseOrderApi";
import { PO_STATUSES } from "../../types/purchaseOrder";
import type { PaginatedPurchaseOrders, PurchaseOrderFilters } from "../../types/purchaseOrder";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const LIMIT = 10;
const STEPS = ["Finance Entry", "Verification", "Admin Review", "Approval", "Payment Ready", "Paid"];

export default function PaymentApproval() {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);

  const [data, setData] = useState<PaginatedPurchaseOrders | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [reviewingPO, setReviewingPO] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    const t = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [search]);

  useEffect(() => {
    setPage(1);
  }, [status]);

  useEffect(() => {
    const filters: PurchaseOrderFilters = {
      status: status || undefined,
      search: debouncedSearch || undefined,
    };
    setLoading(true);
    setError(null);
    getPurchaseOrders(filters, page, LIMIT)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Something went wrong"))
      .finally(() => setLoading(false));
  }, [status, debouncedSearch, page, refreshKey]);

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportPurchaseOrders({ status: status || undefined, search: debouncedSearch || undefined });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to export");
    } finally {
      setExporting(false);
    }
  };

  // Called after Approve / Mark as Paid succeeds, to pull fresh data into this same page/filter.
  const refetch = () => setRefreshKey((k) => k + 1);

  const orders = data?.orders ?? [];

  return (
    <DashboardLayout breadcrumb="Payment Approval">
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">Payment Approval</h1>
          <p className="text-gray-500 text-sm mt-1">
            {data?.total ?? 0} entries awaiting admin review. Open any record to verify margin and
            approve payment.
          </p>
        </div>

        {/* Static workflow legend */}
        <div className="flex items-center mb-6 border border-gray-200 rounded-xl px-6 py-4">
          {STEPS.map((step, i) => {
            const done = i < 2;
            const active = i === 2;
            return (
              <div key={step} className="flex items-center flex-1 last:flex-none">
                <div className="flex flex-col items-center gap-1">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium ${
                      done
                        ? "bg-black text-white"
                        : active
                        ? "border-2 border-black text-black"
                        : "border border-gray-300 text-gray-400"
                    }`}
                  >
                    {done ? <Check size={14} /> : i + 1}
                  </div>
                  <span className="text-[11px] text-gray-500 whitespace-nowrap">{step}</span>
                </div>
                {i < STEPS.length - 1 && (
                  <div className={`flex-1 h-px mx-2 mb-5 ${done ? "bg-black" : "bg-gray-200"}`} />
                )}
              </div>
            );
          })}
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
                  <th className="px-4 py-3 font-medium">PO NUMBER</th>
                  <th className="px-4 py-3 font-medium">VENDOR</th>
                  <th className="px-4 py-3 font-medium">INVOICE</th>
                  <th className="px-4 py-3 font-medium text-right">PAYABLE</th>
                  <th className="px-4 py-3 font-medium text-right">MARGIN</th>
                  <th className="px-4 py-3 font-medium">DUE DATE</th>
                  <th className="px-4 py-3 font-medium">STATUS</th>
                  <th className="px-4 py-3 font-medium text-right">ACTION</th>
                </tr>
              </thead>
              <tbody>
                {loading && (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-gray-400">
                      Loading…
                    </td>
                  </tr>
                )}
                {!loading && error && (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-red-600">
                      {error}
                    </td>
                  </tr>
                )}
                {!loading && !error && orders.length === 0 && (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-gray-400">
                      Nothing here right now.
                    </td>
                  </tr>
                )}
                {!loading &&
                  !error &&
                  orders.map((po) => (
                    <tr key={po.po_number} className="border-t border-gray-100">
                      <td className="px-4 py-3 font-medium">{po.po_number}</td>
                      <td className="px-4 py-3">{po.vendor_name}</td>
                      <td className="px-4 py-3 text-gray-600">{po.invoice_no ?? "—"}</td>
                      <td className="px-4 py-3 text-right">{currency.format(po.landing_cost)}</td>
                      <td
                        className={`px-4 py-3 text-right font-medium ${
                          po.gross_margin_pct < 0 ? "text-red-600" : "text-gray-900"
                        }`}
                      >
                        {po.gross_margin_pct.toFixed(1)}%
                      </td>
                      <td className="px-4 py-3 text-gray-600">{po.expected_delivery_date ?? "—"}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={po.payment_status === "Paid" ? "Paid" : po.status} />
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => setReviewingPO(po.po_number)}
                          className="text-sm px-3 py-1.5 rounded-lg bg-black text-white hover:bg-gray-800"
                        >
                          Review
                        </button>
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
                  className="border border-gray-200 rounded-lg px-3 py-1.5 disabled:opacity-40"
                >
                  Prev
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(data.total_pages, p + 1))}
                  disabled={page >= data.total_pages}
                  className="border border-gray-200 rounded-lg px-3 py-1.5 disabled:opacity-40"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {reviewingPO && (
        <PaymentApprovalDrawer
          poNumber={reviewingPO}
          onClose={() => setReviewingPO(null)}
          onActionComplete={refetch}
        />
      )}
    </DashboardLayout>
  );
}