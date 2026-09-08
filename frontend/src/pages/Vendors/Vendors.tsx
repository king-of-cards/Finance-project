import { useEffect, useState } from "react";
import { Plus, Pencil } from "lucide-react";
import DashboardLayout from "../../layouts/DashboardLayout";
import { getVendors, createVendor, updateVendor } from "../../api/vendorApi";
import { PAYMENT_TERMS } from "../../types/vendor";
import type { Vendor, CreateVendorRequest, UpdateVendorRequest } from "../../types/vendor";

function initials(name: string) {
  return name.split(" ").filter(Boolean).map((w) => w[0]).join("").slice(0, 2).toUpperCase();
}

function VendorCard({ vendor, onEdit }: { vendor: Vendor; onEdit: (v: Vendor) => void }) {
  return (
    <div className="border border-gray-200 rounded-xl p-5 relative">
      <button
        onClick={() => onEdit(vendor)}
        className="absolute top-4 right-4 text-gray-300 hover:text-gray-600"
        title="Edit vendor"
      >
        <Pencil size={15} />
      </button>

      <div className="flex justify-between items-start mb-3 pr-6">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-black text-white flex items-center justify-center text-sm font-medium">
            {initials(vendor.name)}
          </div>
          <div>
            <p className="font-medium">{vendor.name}</p>
            <p className="text-xs text-gray-400">
              {vendor.vendor_id} {vendor.city ? `· ${vendor.city}` : ""}
            </p>
          </div>
        </div>
        <span className="flex items-center gap-1 text-xs bg-gray-100 text-gray-700 px-2.5 py-1 rounded-full">
          <span className="w-1.5 h-1.5 rounded-full bg-green-500" /> {vendor.status}
        </span>
      </div>

      <p className="text-sm text-gray-600 mb-4">
        {vendor.contact_person ?? "—"} {vendor.phone ? `· ${vendor.phone}` : ""}
        <br />
        GST {vendor.gst_number ?? "—"}
      </p>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <p className="font-semibold">—</p>
          <p className="text-[11px] text-gray-400">TOTAL PROCUREMENT</p>
        </div>
        <div>
          <p className="font-semibold">—</p>
          <p className="text-[11px] text-gray-400">AVG MARGIN</p>
        </div>
        <div>
          <p className="font-semibold">—</p>
          <p className="text-[11px] text-gray-400">COMPLETED</p>
        </div>
        <div>
          <p className="font-semibold">{vendor.avg_delivery_days}d</p>
          <p className="text-[11px] text-gray-400">AVG DELIVERY</p>
        </div>
      </div>

      <div className="flex justify-between items-center border-t pt-3">
        <span className="text-sm">
          {"★".repeat(vendor.rating ?? 0)}
          {"☆".repeat(5 - (vendor.rating ?? 0))}
        </span>
        <span className="text-xs bg-gray-100 px-2 py-1 rounded-md">{vendor.payment_terms}</span>
      </div>
    </div>
  );
}

export default function Vendors() {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(true);

  const [showModal, setShowModal] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<CreateVendorRequest>({
    name: "",
    gst_number: "",
    pan_number: "",
    contact_person: "",
    phone: "",
    email: "",
    payment_terms: "Net 15",
    address: "",
  });

  const [editingVendor, setEditingVendor] = useState<Vendor | null>(null);
  const [editForm, setEditForm] = useState<UpdateVendorRequest>({});
  const [editSubmitting, setEditSubmitting] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const loadVendors = () => {
    setLoading(true);
    getVendors()
      .then(setVendors)
      .finally(() => setLoading(false));
  };

  useEffect(loadVendors, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await createVendor(form);
      setShowModal(false);
      setForm({
        name: "",
        gst_number: "",
        pan_number: "",
        contact_person: "",
        phone: "",
        email: "",
        payment_terms: "Net 15",
        address: "",
      });
      loadVendors();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setSubmitting(false);
    }
  };

  const openEdit = (vendor: Vendor) => {
    setEditingVendor(vendor);
    setEditForm({
      name: vendor.name,
      city: vendor.city ?? "",
      contact_person: vendor.contact_person ?? "",
      phone: vendor.phone ?? "",
      email: vendor.email ?? "",
      address: vendor.address ?? "",
      gst_number: vendor.gst_number ?? "",
      pan_number: vendor.pan_number ?? "",
      payment_terms: vendor.payment_terms,
      status: vendor.status,
      rating: vendor.rating ?? undefined,
    });
  };

  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingVendor) return;
    setEditSubmitting(true);
    setEditError(null);
    try {
      await updateVendor(editingVendor.vendor_id, editForm);
      setEditingVendor(null);
      loadVendors();
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setEditSubmitting(false);
    }
  };

  return (
    <DashboardLayout breadcrumb="Vendors">
      <div className="p-6">
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-2xl font-semibold">Vendors</h1>
            <p className="text-gray-500 text-sm mt-1">
              {vendors.length} card suppliers with performance and payment terms.
            </p>
          </div>
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-2 bg-black text-white text-sm px-4 py-2 rounded-lg hover:bg-gray-800"
          >
            <Plus size={16} /> Create Vendor
          </button>
        </div>

        {loading ? (
          <p className="text-gray-400 text-sm">Loading vendors…</p>
        ) : vendors.length === 0 ? (
          <p className="text-gray-400 text-sm">No vendors yet. Click "Create Vendor" to add one.</p>
        ) : (
          <div className="grid grid-cols-3 gap-4">
            {vendors.map((v) => (
              <VendorCard key={v.vendor_id} vendor={v} onEdit={openEdit} />
            ))}
          </div>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl w-full max-w-lg p-6">
            <div className="flex justify-between items-start mb-1">
              <h2 className="text-lg font-semibold">Create Vendor</h2>
              <button onClick={() => setShowModal(false)} className="text-gray-400">
                ✕
              </button>
            </div>
            <p className="text-sm text-gray-500 mb-4">Add a new card supplier to the master.</p>

            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label className="text-xs text-gray-600">Vendor Name *</label>
                <input
                  required
                  placeholder="Business name"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-600">GST Number</label>
                  <input
                    placeholder="22AAAAA0000A1Z5"
                    value={form.gst_number}
                    onChange={(e) => setForm({ ...form, gst_number: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
                <div>
                  <label className="text-xs text-gray-600">PAN</label>
                  <input
                    placeholder="AAAAA0000A"
                    value={form.pan_number}
                    onChange={(e) => setForm({ ...form, pan_number: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-600">Contact Person</label>
                  <input
                    value={form.contact_person}
                    onChange={(e) => setForm({ ...form, contact_person: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
                <div>
                  <label className="text-xs text-gray-600">Phone</label>
                  <input
                    placeholder="+91 ..."
                    value={form.phone}
                    onChange={(e) => setForm({ ...form, phone: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-600">Email</label>
                  <input
                    type="email"
                    value={form.email}
                    onChange={(e) => setForm({ ...form, email: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
                <div>
                  <label className="text-xs text-gray-600">Payment Terms</label>
                  <select
                    value={form.payment_terms}
                    onChange={(e) => setForm({ ...form, payment_terms: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  >
                    {PAYMENT_TERMS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div>
                <label className="text-xs text-gray-600">Address</label>
                <textarea
                  value={form.address}
                  onChange={(e) => setForm({ ...form, address: e.target.value })}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  rows={3}
                />
              </div>

              {error && <p className="text-red-600 text-sm">{error}</p>}

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-sm rounded-lg border border-gray-300"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="px-4 py-2 text-sm rounded-lg bg-black text-white disabled:opacity-50"
                >
                  {submitting ? "Creating..." : "Create Vendor"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {editingVendor && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl w-full max-w-lg p-6">
            <div className="flex justify-between items-start mb-1">
              <h2 className="text-lg font-semibold">Edit Vendor</h2>
              <button onClick={() => setEditingVendor(null)} className="text-gray-400">
                ✕
              </button>
            </div>
            <p className="text-sm text-gray-500 mb-4">Update {editingVendor.name}'s details.</p>

            <form onSubmit={handleEditSubmit} className="space-y-3">
              <div>
                <label className="text-xs text-gray-600">Vendor Name</label>
                <input
                  value={editForm.name ?? ""}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-600">GST Number</label>
                  <input
                    value={editForm.gst_number ?? ""}
                    onChange={(e) => setEditForm({ ...editForm, gst_number: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
                <div>
                  <label className="text-xs text-gray-600">PAN</label>
                  <input
                    value={editForm.pan_number ?? ""}
                    onChange={(e) => setEditForm({ ...editForm, pan_number: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-600">Contact Person</label>
                  <input
                    value={editForm.contact_person ?? ""}
                    onChange={(e) => setEditForm({ ...editForm, contact_person: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
                <div>
                  <label className="text-xs text-gray-600">Phone</label>
                  <input
                    value={editForm.phone ?? ""}
                    onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-600">Payment Terms</label>
                  <select
                    value={editForm.payment_terms ?? "Net 15"}
                    onChange={(e) => setEditForm({ ...editForm, payment_terms: e.target.value })}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  >
                    {PAYMENT_TERMS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="text-xs text-gray-600">Status</label>
                  <select
                    value={editForm.status ?? "Active"}
                    onChange={(e) =>
                      setEditForm({ ...editForm, status: e.target.value as "Active" | "Inactive" })
                    }
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  >
                    <option value="Active">Active</option>
                    <option value="Inactive">Inactive</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="text-xs text-gray-600">Address</label>
                <textarea
                  value={editForm.address ?? ""}
                  onChange={(e) => setEditForm({ ...editForm, address: e.target.value })}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mt-1"
                  rows={3}
                />
              </div>

              {editError && <p className="text-red-600 text-sm">{editError}</p>}

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setEditingVendor(null)}
                  className="px-4 py-2 text-sm rounded-lg border border-gray-300"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={editSubmitting}
                  className="px-4 py-2 text-sm rounded-lg bg-black text-white disabled:opacity-50"
                >
                  {editSubmitting ? "Saving..." : "Save Changes"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}