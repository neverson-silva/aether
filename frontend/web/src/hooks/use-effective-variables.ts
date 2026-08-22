import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ResolvedVariable } from "../api/types";

export function useEffectiveVariables(appID: string) {
  return useQuery({
    queryKey: ["apps", appID, "effective-variables"],
    queryFn: () => apiGet<{ variables: ResolvedVariable[] }>(`/api/v1/apps/${appID}/variables/effective`),
    enabled: !!appID,
  });
}
