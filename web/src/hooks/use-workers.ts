import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Worker } from "../api/types";

export function useWorkers(appID: string) {
  return useQuery({
    queryKey: ["workers", appID],
    queryFn: () => apiGet<Worker[]>(`/api/v1/apps/${appID}/workers`),
    enabled: !!appID,
  });
}
