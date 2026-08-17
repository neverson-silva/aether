import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Domain } from "../api/types";
import { qk } from "./query-keys";

export function useDomains(appID: string) {
  return useQuery({
    queryKey: qk.domains(appID),
    queryFn: () => apiGet<Domain[]>(`/api/v1/apps/${appID}/domains`),
    enabled: !!appID,
  });
}
