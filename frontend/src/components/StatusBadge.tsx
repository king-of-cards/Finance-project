const DOT_COLORS: Record<string, string> = {
  "PO Raised": "bg-gray-400",
  "Waiting for Vendor": "bg-amber-400",
  Received: "bg-blue-400",
  Verified: "bg-emerald-500",
  "Pending Approval": "bg-amber-500",
  Approved: "bg-emerald-500",
  Rejected: "bg-red-400",
  Cancelled: "bg-gray-400",
  Hold: "bg-orange-400",
};

export default function StatusBadge({ status }: { status: string }) {
  if (status === "Paid") {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs bg-gray-900 text-white px-2.5 py-1 rounded-full">
        <span className="w-1.5 h-1.5 rounded-full bg-white" />
        Paid
      </span>
    );
  }

  const dot = DOT_COLORS[status] ?? "bg-gray-400";

  return (
    <span className="inline-flex items-center gap-1.5 text-xs bg-gray-100 text-gray-700 px-2.5 py-1 rounded-full">
      <span className={`w-1.5 h-1.5 rounded-full ${dot}`} />
      {status}
    </span>
  );
}