import type {
  Vendor,
  CreateVendorRequest,
  UpdateVendorRequest,
} from "../types/vendor";
import { apiJson } from "./httpClient";

export async function getVendors(): Promise<Vendor[]> {
  const data = await apiJson<{ vendors?: Vendor[] }>(
    "/vendors",
    {},
    "Failed to fetch vendors"
  );
  return data.vendors ?? [];
}

export async function createVendor(
  payload: CreateVendorRequest
): Promise<Vendor> {
  return apiJson<Vendor>(
    "/vendors",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    "Failed to create vendor"
  );
}

export async function updateVendor(
  vendorId: string,
  payload: UpdateVendorRequest
): Promise<Vendor> {
  return apiJson<Vendor>(
    `/vendors/${vendorId}`,
    {
      method: "PATCH",
      body: JSON.stringify(payload),
    },
    "Failed to update vendor"
  );
}
