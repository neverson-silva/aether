import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export function useAppCompose(appID: string, enabled = true) {
  return useQuery({
    queryKey: ["app-compose", appID],
    enabled: !!appID && enabled,
    queryFn: () => apiGet<{ compose: string }>(`/api/v1/apps/${appID}/compose`),
  });
}
