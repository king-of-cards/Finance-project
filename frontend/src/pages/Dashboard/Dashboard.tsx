import { useEffect, useState } from "react";
import { Download } from "lucide-react";
import DashboardLayout from "../../layouts/DashboardLayout";
import StatCard from "../../components/StatCard";
import StatusBadge from "../../components/StatusBadge";
import SimpleBarChart from "../../components/SimpleBarChart";
import { getDashboardOverview } from "../../api/dashboardApi";
import { exportReport } from "../../api/reportsApi";
import { getVendors } from "../../api/vendorApi";
import type { DashboardOverview } from "../../types/dashboard";
import type { Vendor } from "../../types/vendor";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

export default function Dashboard() {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [vendorId, setVendorId] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [appliedFilters, setAppliedFilters] = useState({ vendorId: "", from: "", to: "" });

  const [data, setData] = useState<DashboardOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    getVendors().then(setVendors).catch(() => setVendors([]));
  }, []);

  useEffect(() => {
    setLoading(true);
    setError(null);
    getDashboardOverview(appliedFilters)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load dashboard"))
      .finally(() => setLoading(false));
  }, [appliedFilters]);

  const applyFilters = () => setAppliedFilters({ vendorId, from, to });
  const resetFilters = () => {
    setVendorId("");
    setFrom("");
    setTo("");
    setAppliedFilters({ vendorId: "", from: "", to: "" });
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportReport("monthly", "csv", appliedFilters);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to export");
    } finally {
      setExporting(false);
    }
  };

  const k = data?.kpis;
  const stats = [
    { label: "Total Orders", value: k ? k.total_orders.toLocaleString("en-IN") : "—" },
    { label: "Total Procurement Value", value: k ? currency.format(k.total_spend) : "—" },
    { label: "Pending Approvals", value: k ? k.pending_approvals.toLocaleString("en-IN") : "—" },
    { label: "Active Vendors", value: k ? k.active_vendors.toLocaleString("en-IN") : "—" },
    { label: "Average Gross Margin %", value: k ? `${k.avg_margin_pct.toFixed(1)}%` : "—" },
    { label: "Overdue Orders", value: k ? k.overdue_orders.toLocaleString("en-IN") : "—" },
    { label: "Total Paid", value: k ? currency.format(k.total_paid) : "—" },
    { label: "Avg Vendor Turnaround", value: k ? `${k.avg_delivery_days.toFixed(1)} days` : "—" },
  ];

  return (
    <DashboardLayout breadcrumb="Dashboard">
      <div className="p-6">
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-2xl font-semibold">Procurement Overview</h1>
            <p className="text-gray-500 text-sm mt-1">
              Real-time landing cost, margins and approvals across vendors.
            </p>
          </div>
          <button
            onClick={handleExport}
            disabled={exporting}
            className="flex items-center gap-2 border border-gray-200 rounded-lg px-4 py-2 text-sm hover:bg-gray-50 disabled:opacity-50"
          >
            <Download size={14} /> {exporting ? "Exporting…" : "Export Report"}
          </button>
        </div>

        <div className="flex flex-wrap items-end gap-3 mb-6 border border-gray-200 rounded-xl p-4">
          <div>
            <label className="text-xs text-gray-400">VENDOR</label>
            <select
              value={vendorId}
              onChange={(e) => setVendorId(e.target.value)}
              className="mt-1 block text-sm border border-gray-200 rounded-lg px-3 py-2"
            >
              <option value="">All vendors</option>
              {vendors.map((v) => (
                <option key={v.vendor_id} value={v.vendor_id}>
                  {v.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs text-gray-400">FROM</label>
            <input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="mt-1 block text-sm border border-gray-200 rounded-lg px-3 py-2"
            />
          </div>
          <div>
            <label className="text-xs text-gray-400">TO</label>
            <input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="mt-1 block text-sm border border-gray-200 rounded-lg px-3 py-2"
            />
          </div>
          <button onClick={applyFilters} className="text-sm bg-black text-white rounded-lg px-4 py-2">
            Apply
          </button>
          <button onClick={resetFilters} className="text-sm border border-gray-200 rounded-lg px-4 py-2">
            Reset
          </button>
        </div>

        {error && (
          <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mb-4">
            {error}
          </p>
        )}

        <div className="grid grid-cols-4 gap-4 mb-6">
          {stats.map((s) => (
            <StatCard key={s.label} {...s} />
          ))}
        </div>

        {data && (
          <>
            <div className="grid grid-cols-2 gap-4 mb-6">
              <SimpleBarChart
                title="Monthly Procurement Value"
                points={data.monthly_value}
                formatValue={(v) => currency.format(v)}
              />
              <SimpleBarChart
                title="Vendor-wise Spend"
                points={data.vendor_wise_spend}
                formatValue={(v) => currency.format(v)}
              />
            </div>

            <div className="border border-gray-200 rounded-xl overflow-hidden">
              <p className="text-sm font-semibold px-4 py-3 border-b border-gray-200">Recent Orders</p>
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-[11px] tracking-wide text-gray-400">
                    <th className="px-4 py-3 font-medium">PO NUMBER</th>
                    <th className="px-4 py-3 font-medium">VENDOR</th>
                    <th className="px-4 py-3 font-medium text-right">LANDING COST</th>
                    <th className="px-4 py-3 font-medium text-right">MARGIN</th>
                    <th className="px-4 py-3 font-medium">STATUS</th>
                  </tr>
                </thead>
                <tbody>
                  {loading && (
                    <tr>
                      <td colSpan={5} className="px-4 py-6 text-center text-gray-400">
                        Loading…
                      </td>
                    </tr>
                  )}
                  {!loading && data.recent_orders.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-4 py-6 text-center text-gray-400">
                        No recent orders.
                      </td>
                    </tr>
                  )}
                  {!loading &&
                    data.recent_orders.map((po) => (
                      <tr key={po.po_number} className="border-t border-gray-100">
                        <td className="px-4 py-3 font-medium">{po.po_number}</td>
                        <td className="px-4 py-3">{po.vendor_name}</td>
                        <td className="px-4 py-3 text-right">{currency.format(po.landing_cost)}</td>
                        <td className="px-4 py-3 text-right">{po.gross_margin_pct.toFixed(1)}%</td>
                        <td className="px-4 py-3">
                          <StatusBadge status={po.status} />
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}