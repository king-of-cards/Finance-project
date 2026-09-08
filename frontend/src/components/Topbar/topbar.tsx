// src/components/Topbar.tsx
import { Search, RefreshCcw, Bell, Download } from 'lucide-react'

interface TopbarProps {
  breadcrumb?: string
}

export default function Topbar({ breadcrumb = 'Dashboard' }: TopbarProps) {
  return (
    <header className="h-16 border-b border-gray-200 flex items-center justify-between px-6">
      <div className="flex items-center gap-2 text-sm text-gray-500">
        King of Cards <span>/</span> <span className="text-gray-900 font-medium">{breadcrumb}</span>
      </div>
      <div className="flex-1 max-w-md mx-6 relative">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input
          placeholder="Search PO, invoice, vendor, SKU..."
          className="w-full pl-9 pr-3 py-2 text-sm border border-gray-200 rounded-md"
        />
      </div>
      <div className="flex items-center gap-4 text-sm text-gray-600">
        <span>
          {new Date().toLocaleDateString('en-US', {
            weekday: 'short',
            day: '2-digit',
            month: 'short',
            year: 'numeric',
          })}
        </span>
        <RefreshCcw size={16} />
        <Bell size={16} />
        <button className="flex items-center gap-1 bg-gray-900 text-white text-sm px-3 py-1.5 rounded-md">
          <Download size={14} /> Export
        </button>
      </div>
    </header>
  )
}