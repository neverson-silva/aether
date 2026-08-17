import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export function useDeploymentCompose(appID: string, depID: string | null) {
  return useQuery({
    queryKey: ["dep-compose", appID, depID],
    enabled: !!depID,
    queryFn: () => apiGet<{ number: number; hash: string; compose: string }>(`/api/v1/apps/${appID}/deployments/${depID}/compose`),
  });
}
