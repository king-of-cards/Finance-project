import type { ChargeType } from "../types/chargeType";
import { apiJson } from "./httpClient";
 
export async function getChargeTypes(): Promise<ChargeType[]> {
  return apiJson<ChargeType[]>("/charge-types", {}, "Failed to fetch charge types");
}