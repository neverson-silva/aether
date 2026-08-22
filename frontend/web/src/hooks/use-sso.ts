import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { OIDCProvider } from "./types";

export function useSSO() {
  return useQuery({ queryKey: ["sso"], queryFn: () => apiGet<OIDCProvider[]>("/api/v1/sso") });
}
