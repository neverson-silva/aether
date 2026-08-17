import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Deployment } from "../api/types";
import { qk } from "./query-keys";

export function useDeployments(appID: string) {
  return useQuery({
    queryKey: qk.deployments(appID),
    queryFn: () => apiGet<Deployment[]>(`/api/v1/apps/${appID}/deployments`),
    enabled: !!appID,
  });
}
