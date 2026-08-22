import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ApiKey } from "../api/types";
import { qk } from "./query-keys";

export function useApiKeys() {
  return useQuery({ queryKey: qk.keys, queryFn: () => apiGet<ApiKey[]>("/api/v1/api-keys") });
}
