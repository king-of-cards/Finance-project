import { useEffect, useState } from "react";
import DashboardLayout from "../../layouts/DashboardLayout";
import StatCard from "../../components/StatCard";
import SimpleBarChart from "../../components/SimpleBarChart";
import { getAnalyticsOverview } from "../../api/analyticsApi";
import { getDashboardOverview } from "../../api/dashboardApi";
import type { AnalyticsOverview } from "../../types/analytics";
import type { ChartPoint } from "../../types/dashboard";

const currency = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

export default function Analytics() {
  const [data, setData] = useState<AnalyticsOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Vendor Comparison, Transportation Trend, and Average Delivery Time aren't
  // returned by /analytics/overview — they only exist on the Dashboard
  // endpoint's vendor_wise_spend, transport_trend, and vendor_performance
  // fields. Fetched here as a workaround until Analytics returns them directly.
  const [vendorSpend, setVendorSpend] = useState<ChartPoint[]>([]);
  const [transportTrend, setTransportTrend] = useState<ChartPoint[]>([]);
  const [vendorPerformance, setVendorPerformance] = useState<ChartPoint[]>([]);

  useEffect(() => {
    setLoading(true);
    setError(null);
    getAnalyticsOverview({})
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load analytics"))
      .finally(() => setLoading(false));

    getDashboardOverview({})
      .then((d) => {
        setVendorSpend(d.vendor_wise_spend ?? []);
        setTransportTrend(d.transport_trend ?? []);
        setVendorPerformance(d.vendor_performance ?? []);
      })
      .catch(() => {
        setVendorSpend([]);
        setTransportTrend([]);
        setVendorPerformance([]);
      });
  }, []);

  const k = data?.kpis;

  // Highest/lowest margin vendor aren't standalone KPI fields — derived from
  // the margin_by_vendor chart array the endpoint already returns.
  const marginVendors = data?.margin_by_vendor ?? [];
  const highestMarginVendor = marginVendors.length
    ? marginVendors.reduce((a, b) => (b.value > a.value ? b : a))
    : null;
  const lowestMarginVendor = marginVendors.length
    ? marginVendors.reduce((a, b) => (b.value < a.value ? b : a))
    : null;

  // "Most Delayed Vendor" also isn't a standalone field — derived from the
  // per-vendor avg delivery days in vendor_performance (Dashboard endpoint).
  const mostDelayedVendor = vendorPerformance.length
    ? vendorPerformance.reduce((a, b) => (b.value > a.value ? b : a))
    : null;

  const stats = [
    { label: "Total Procurement", value: k ? currency.format(k.total_spend) : "—" },
    { label: "Vendor Spend", value: k ? currency.format(k.total_base_cost) : "—" },
    { label: "Transportation Spend", value: k ? currency.format(k.total_transport_cost) : "—" },
    { label: "Handling Spend", value: k ? currency.format(k.total_handling_cost) : "—" },
    { label: "Packaging Spend", value: k ? currency.format(k.total_packaging_cost) : "—" },
    { label: "GST Paid", value: k ? currency.format(k.total_gst) : "—" },
    { label: "Avg Landing Cost / unit", value: k ? currency.format(k.avg_landing_cost_per_unit) : "—" },
    { label: "Avg Gross Margin", value: k ? `${k.avg_margin_pct.toFixed(1)}%` : "—" },
    { label: "Highest Margin Vendor", value: highestMarginVendor?.label ?? "—" },
    { label: "Lowest Margin Vendor", value: lowestMarginVendor?.label ?? "—" },
    { label: "Most Delayed Vendor", value: mostDelayedVendor?.label ?? "—" },
  ];

  return (
    <DashboardLayout breadcrumb="Analytics">
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">Analytics</h1>
          <p className="text-gray-500 text-sm mt-1">
            Procurement analytics across vendors and time.
          </p>
        </div>

        {error && (
          <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mb-4">
            {error}
          </p>
        )}
        {loading && <p className="text-sm text-gray-400 mb-4">Loading…</p>}

        <div className="grid grid-cols-4 gap-4 mb-6">
          {stats.map((s) => (
            <StatCard key={s.label} {...s} />
          ))}
        </div>

        {data && (
          <div className="grid grid-cols-2 gap-4">
            <SimpleBarChart
              title="Vendor Comparison"
              points={vendorSpend}
              formatValue={(v) => currency.format(v)}
            />
            <SimpleBarChart
              title="Monthly Procurement (units)"
              points={data.procurement_by_month}
              formatValue={(v) => v.toLocaleString("en-IN")}
            />
            <SimpleBarChart
              title="Transportation Trend"
              points={transportTrend}
              formatValue={(v) => currency.format(v)}
            />
            <SimpleBarChart
              title="Average Delivery Time (days per vendor)"
              points={vendorPerformance}
              formatValue={(v) => `${v.toFixed(1)}d`}
            />
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}