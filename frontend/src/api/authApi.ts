import type { LoginRequest, LoginResponse } from "../types/auth";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;

export async function loginUser(
  userId: string,
  password: string
): Promise<LoginResponse> {
  const requestBody: LoginRequest = {
    user_id: userId,
    password: password,
  };

  const response = await fetch(`${BASE_URL}/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(requestBody),
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.error || "Login failed");
  }

  const data: LoginResponse = await response.json();
  return data;
}