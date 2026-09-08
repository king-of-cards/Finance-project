// src/components/StatCard.tsx
interface StatCardProps {
  label: string
  value: string
  trend?: 'up' | 'down'
  icon?: React.ReactNode
}

export default function StatCard({ label, value, trend, icon }: StatCardProps) {
  return (
    <div className="border border-gray-200 rounded-lg p-4">
      <div className="flex justify-between items-start text-sm text-gray-500">
        <span>{label}</span>
        {icon}
      </div>
      <p className="text-2xl font-semibold mt-2">{value}</p>
      {trend && (
        <div className="flex items-center gap-1 mt-2 text-xs text-gray-400">
          <span className={trend === 'up' ? 'text-green-600' : 'text-red-500'}>
            {trend === 'up' ? '▲' : '▼'}
          </span>
          vs last month
        </div>
      )}
    </div>
  )
}