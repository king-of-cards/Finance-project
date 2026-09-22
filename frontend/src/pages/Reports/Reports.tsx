import { useState } from "react";
import { FileText } from "lucide-react";
import DashboardLayout from "../../layouts/DashboardLayout";
import { exportReport } from "../../api/reportsApi";
import type { ReportType, ReportFormat } from "../../api/reportsApi";

interface ReportCard {
  type: ReportType;
  title: string;
  description: string;
}

const REPORT_CARDS: ReportCard[] = [
  { type: "daily", title: "Daily Report", description: "Orders received and entered today." },
  { type: "weekly", title: "Weekly Report", description: "Rolling 7-day procurement summary." },
  { type: "monthly", title: "Monthly Report", description: "Full month landing cost & margin." },
  { type: "vendor", title: "Vendor Report", description: "Spend and performance per vendor." },
  { type: "margin", title: "Margin Report", description: "Gross profit and margin by order." },
  { type: "gst", title: "GST Report", description: "Input GST by slab for filing." },
  { type: "payment", title: "Payment Report", description: "Approved, pending and paid ledger." },
  { type: "procurement", title: "Procurement Report", description: "Complete PO register." },
  {
    type: "inventory-landing-cost",
    title: "Inventory Landing Cost Report",
    description: "Per-SKU landed cost basis.",
  },
];

export default function Reports() {
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleExport = async (type: ReportType, format: ReportFormat) => {
    const key = `${type}-${format}`;
    setPending(key);
    setError(null);
    try {
      await exportReport(type, format, {});
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to export ${type} report`);
    } finally {
      setPending(null);
    }
  };

  return (
    <DashboardLayout breadcrumb="Reports">
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">Reports</h1>
          <p className="text-gray-500 text-sm mt-1">
            Generate and export procurement, margin, GST and payment reports.
          </p>
        </div>

        {error && (
          <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mb-4">
            {error}
          </p>
        )}

        <div className="grid grid-cols-3 gap-4">
          {REPORT_CARDS.map((card) => (
            <div key={card.type} className="border border-gray-200 rounded-xl p-5">
              <div className="w-9 h-9 rounded-lg bg-gray-100 flex items-center justify-center mb-4">
                <FileText size={16} className="text-gray-500" />
              </div>
              <p className="font-semibold mb-1">{card.title}</p>
              <p className="text-sm text-gray-400 mb-4">{card.description}</p>
              <div className="flex gap-2">
                <button
                  onClick={() => handleExport(card.type, "csv")}
                  disabled={pending === `${card.type}-csv`}
                  className="text-sm border border-gray-200 rounded-lg px-3 py-1.5 hover:bg-gray-50 disabled:opacity-50"
                >
                  {pending === `${card.type}-csv` ? "Exporting…" : "Excel / CSV"}
                </button>
                <button
                  onClick={() => handleExport(card.type, "pdf")}
                  disabled={pending === `${card.type}-pdf`}
                  className="text-sm border border-gray-200 rounded-lg px-3 py-1.5 hover:bg-gray-50 disabled:opacity-50"
                >
                  {pending === `${card.type}-pdf` ? "Exporting…" : "Print / PDF"}
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </DashboardLayout>
  );
}