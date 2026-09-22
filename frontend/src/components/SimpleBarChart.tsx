interface Props {
  title: string;
  points: { label: string; value: number }[];
  formatValue?: (v: number) => string;
}

export default function SimpleBarChart({ title, points, formatValue }: Props) {
  const max = Math.max(1, ...points.map((p) => Math.abs(p.value)));
  const format = formatValue ?? ((v: number) => v.toLocaleString("en-IN"));

  return (
    <div className="border border-gray-200 rounded-xl p-4">
      <p className="text-sm font-semibold mb-4">{title}</p>
      {points.length === 0 && <p className="text-sm text-gray-400">No data for this range.</p>}
      <div className="space-y-2">
        {points.map((p) => (
          <div key={p.label} className="flex items-center gap-3 text-xs">
            <span className="w-24 shrink-0 text-gray-500 truncate">{p.label}</span>
            <div className="flex-1 bg-gray-100 rounded h-4 overflow-hidden">
              <div
                className="bg-black h-full rounded"
                style={{ width: `${(Math.abs(p.value) / max) * 100}%` }}
              />
            </div>
            <span className="w-20 shrink-0 text-right text-gray-600">{format(p.value)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}