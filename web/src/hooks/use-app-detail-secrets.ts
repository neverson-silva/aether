import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AppDetail } from "../api/types";
import { qk } from "./query-keys";

export function useAppDetailSecrets(id: string, enabled: boolean) {
  return useQuery({
    queryKey: [...qk.app(id), "secrets"],
    queryFn: () => apiGet<AppDetail>(`/api/v1/apps/${id}?secrets=1`),
    enabled,
  });
}
