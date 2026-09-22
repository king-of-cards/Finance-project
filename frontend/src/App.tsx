import { BrowserRouter, Routes, Route } from "react-router-dom";
import Login from "./pages/Login/Login";
import Dashboard from "./pages/Dashboard/Dashboard";
import Vendors from "./pages/Vendors/Vendors";
import VendorEntries from "./pages/VendorEntries/VendorEntries";
import ProtectedRoute from "./components/ProtectedRoute";
import PaymentApproval from "./pages/PaymentApproval/PaymentApproval";
import Analytics from "./pages/Analytics/Analytics";
import Reports from "./pages/Reports/Reports";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />

        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/vendor-entries" element={<VendorEntries />} />
          {/* TODO: replace with real pages as they're built */}
          <Route path="/payment-approval" element={<PaymentApproval />} />
          <Route path="/vendors" element={<Vendors />} />
          <Route path="/reports" element={<Reports />} />
          <Route path="/analytics" element={<Analytics />} />
          <Route path="/settings" element={<Dashboard />} />
        </Route>

        <Route path="*" element={<Login />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;