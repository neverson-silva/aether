import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { SystemSummary } from "./types";

export function useSystemSummary() {
  return useQuery({ queryKey: ["system-summary"], queryFn: () => apiGet<SystemSummary>("/api/v1/system/summary") });
}
