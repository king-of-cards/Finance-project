export interface Vendor {
  vendor_id: string;
  name: string;
  city: string | null;
  contact_person: string | null;
  phone: string | null;
  email: string | null;
  address: string | null;
  gst_number: string | null;
  pan_number: string | null;
  payment_terms: string;
  avg_delivery_days: number;
  rating: number | null;
  status: "Active" | "Inactive";
}

export const PAYMENT_TERMS = ["Net 15", "Net 30", "Net 45", "Advance 50%", "On Delivery"] as const;

export interface CreateVendorRequest {
  name: string;
  city?: string;
  contact_person?: string;
  phone?: string;
  email?: string;
  address?: string;
  gst_number?: string;
  pan_number?: string;
  payment_terms?: string;
}

export interface UpdateVendorRequest {
  name?: string;
  city?: string;
  contact_person?: string;
  phone?: string;
  email?: string;
  address?: string;
  gst_number?: string;
  pan_number?: string;
  payment_terms?: string;
  status?: "Active" | "Inactive";
  rating?: number;
}