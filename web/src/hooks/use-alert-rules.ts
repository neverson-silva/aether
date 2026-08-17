import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AlertRule } from "./types";

export function useAlertRules() {
  return useQuery({ queryKey: ["alert-rules"], queryFn: () => apiGet<AlertRule[]>("/api/v1/alerts/rules") });
}
