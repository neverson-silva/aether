import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { App } from "../api/types";
import { qk } from "./query-keys";

export function useApps() {
  return useQuery({ queryKey: qk.apps, queryFn: () => apiGet<App[]>("/api/v1/apps") });
}
