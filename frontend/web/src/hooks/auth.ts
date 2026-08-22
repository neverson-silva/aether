import { redirect } from "@tanstack/react-router";
import { apiGet } from "../api/client";
import type { Me } from "../api/types";
import { useAuthStore } from "../stores/auth";

// requireAuth guards protected routes. The authenticated user lives in the
// auth store, populated right after login; navigating to a protected route
// reads the store synchronously with no request. The session cookie is only
// validated once per cold load (store empty), then cached in the store.
export async function requireAuth(): Promise<Me> {
  const { user } = useAuthStore.getState();
  if (user) {
    return user;
  }
  try {
    const me = await apiGet<Me>("/api/v1/me");
    useAuthStore.getState().setUser(me);
    return me;
  } catch {
    throw redirect({ to: "/login" });
  }
}