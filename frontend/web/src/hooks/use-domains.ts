import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Domain } from "../api/types";
import { qk } from "./query-keys";

export function useDomains(kind: string, id: string) {
  return useQuery({
    queryKey: qk.domains(kind, id),
    queryFn: () => apiGet<Domain[]>(`/api/v1/${kind}/${id}/domains`),
    enabled: !!id,
  });
}
