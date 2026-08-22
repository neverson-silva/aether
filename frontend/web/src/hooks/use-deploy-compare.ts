import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { DeployCompare } from "./types";

export function useDeployCompare(appID: string, a: string | null, b: string | null) {
  return useQuery({
    queryKey: ["deploy-compare", appID, a, b],
    enabled: !!(a && b),
    queryFn: () => apiGet<DeployCompare>(`/api/v1/apps/${appID}/deployments/compare?a=${a}&b=${b}`),
  });
}
