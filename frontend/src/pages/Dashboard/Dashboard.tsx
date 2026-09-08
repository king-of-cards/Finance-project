// src/pages/Dashboard.tsx
import StatCard from '../../components/StatCard'
import DashboardLayout from '../../layouts/DashboardLayout'

const stats = [
  { label: 'Total Vendor Entries', value: '—' },
  { label: 'Pending Payment Approvals', value: '—' },
  { label: "Today's Received Orders", value: '—' },
  { label: 'SKUs Procured', value: '—' },
  { label: 'Total Procurement Value', value: '—' },
  { label: 'Average Landing Cost %', value: '—' },
  { label: 'Average Gross Margin %', value: '—' },
  { label: 'Avg Vendor Turnaround', value: '—' },
]

export default function Dashboard() {
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
          {/* Reports + New Vendor Entry buttons */}
        </div>

        {/* filter bar: vendor select, from/to date, apply, reset, download */}

        <div className="grid grid-cols-4 gap-4">
          {stats.map((s) => (
            <StatCard key={s.label} {...s} />
          ))}
        </div>
      </div>
    </DashboardLayout>
  )
}