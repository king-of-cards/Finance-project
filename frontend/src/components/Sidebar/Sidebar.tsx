// src/components/Sidebar/Sidebar.tsx
import { useNavigate, useLocation, Link } from "react-router-dom";
import {
  LayoutGrid,
  PenLine,
  CheckSquare,
  Users,
  FileText,
  BarChart2,
  Settings,
  LogOut,
} from "lucide-react";
import type { User } from "../../types/auth";

interface NavLink {
  icon: typeof LayoutGrid;
  label: string;
  path: string;
  badge?: number;
}

const opsLinks: NavLink[] = [
  { icon: LayoutGrid, label: "Dashboard", path: "/dashboard" },
  { icon: PenLine, label: "Vendor Entries", path: "/vendor-entries" },
  { icon: CheckSquare, label: "Payment Approval", path: "/payment-approval" },
  { icon: Users, label: "Vendors", path: "/vendors" },
];

const insightLinks: NavLink[] = [
  { icon: FileText, label: "Reports", path: "/reports" },
  { icon: BarChart2, label: "Analytics", path: "/analytics" },
  { icon: Settings, label: "Settings", path: "/settings" },
];

function getInitials(name: string) {
  return name
    .split(" ")
    .filter(Boolean)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

function getStoredUser(): User | null {
  const raw = localStorage.getItem("user");
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
}

export default function Sidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = getStoredUser();

  const handleLogout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    navigate("/login");
  };

  const renderLink = ({ icon: Icon, label, path, badge }: NavLink) => {
    const isActive = location.pathname === path;
    return (
      <Link
        key={path}
        to={path}
        className={`flex items-center justify-between gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
          isActive
            ? "bg-black text-white font-medium"
            : "text-gray-700 hover:bg-gray-100"
        }`}
      >
        <span className="flex items-center gap-3">
          <Icon size={17} strokeWidth={1.75} />
          {label}
        </span>
        {typeof badge === "number" && (
          <span
            className={`text-xs rounded-full px-2 py-0.5 leading-none ${
              isActive ? "bg-white/20 text-white" : "bg-gray-100 text-gray-600"
            }`}
          >
            {badge}
          </span>
        )}
      </Link>
    );
  };

  return (
    <aside className="w-60 h-screen bg-white text-gray-900 border-r border-gray-200 flex flex-col justify-between shrink-0">
      <div>
        {/* Logo block */}
        <div className="flex items-center gap-2.5 px-5 py-5">
          <div className="w-8 h-8 rounded-md bg-black text-white flex items-center justify-center font-semibold text-sm">
            K
          </div>
          <div className="leading-tight">
            <p className="text-sm font-semibold">King of Cards</p>
            <p className="text-[10px] tracking-wide text-gray-400">PROCUREMENT ERP</p>
          </div>
        </div>

        <div className="border-t border-gray-100" />

        {/* Nav sections */}
        <nav className="px-3 space-y-6 mt-4">
          <div>
            <p className="px-3 mb-2 text-[10px] tracking-wider text-gray-400 font-medium">
              OPERATIONS
            </p>
            <div className="space-y-1">{opsLinks.map(renderLink)}</div>
          </div>
          <div>
            <p className="px-3 mb-2 text-[10px] tracking-wider text-gray-400 font-medium">
              INSIGHTS
            </p>
            <div className="space-y-1">{insightLinks.map(renderLink)}</div>
          </div>
        </nav>
      </div>

      {/* Bottom block: logged-in user + logout */}
      <div className="border-t border-gray-200 p-3">
        <div className="flex items-center gap-2 px-2 py-2">
          <div className="w-8 h-8 rounded-full bg-gray-200 text-gray-700 flex items-center justify-center text-xs shrink-0">
            {user ? getInitials(user.name) : "?"}
          </div>
          <div className="text-sm leading-tight min-w-0">
            <p className="font-medium truncate">{user?.name ?? "Guest"}</p>
            <p className="text-gray-500 text-xs truncate">{user?.role ?? "Not signed in"}</p>
          </div>
        </div>
        <button
          onClick={handleLogout}
          className="w-full flex items-center gap-2 px-2 py-2 mt-1 rounded-md text-sm text-gray-700 border border-gray-200 hover:bg-gray-50 transition-colors"
        >
          <LogOut size={16} /> Log out
        </button>
      </div>
    </aside>
  );
}