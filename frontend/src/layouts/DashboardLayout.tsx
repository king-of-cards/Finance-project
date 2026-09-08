// src/layouts/DashboardLayout.tsx
import type { ReactNode } from "react";
import Sidebar from "../components/Sidebar/Sidebar";
import Topbar from "../components/Topbar/topbar";

export default function DashboardLayout({
  children,
  breadcrumb,
}: {
  children: ReactNode;
  breadcrumb?: string;
}) {
  return (
    <div className="flex">
      <Sidebar />
      <main className="flex-1 min-w-0 min-h-screen">
        <Topbar breadcrumb={breadcrumb} />
        {children}
      </main>
    </div>
  );
}