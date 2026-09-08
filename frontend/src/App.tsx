import { BrowserRouter, Routes, Route } from "react-router-dom";
import Login from "./pages/Login/Login";
import Dashboard from "./pages/Dashboard/Dashboard";
import Vendors from "./pages/Vendors/Vendors";
import VendorEntries from "./pages/VendorEntries/VendorEntries";
import ProtectedRoute from "./components/ProtectedRoute";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />

        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/vendor-entries" element={<VendorEntries />} />
          {/* TODO: replace with real pages as they're built */}
          <Route path="/payment-approval" element={<Dashboard />} />
          <Route path="/vendors" element={<Vendors />} />
          <Route path="/reports" element={<Dashboard />} />
          <Route path="/analytics" element={<Dashboard />} />
          <Route path="/settings" element={<Dashboard />} />
        </Route>

        <Route path="*" element={<Login />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;