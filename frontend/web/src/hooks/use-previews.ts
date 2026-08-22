import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Preview } from "../api/types";

export function usePreviews(appID: string) {
  return useQuery({
    queryKey: ["previews", appID],
    queryFn: () => apiGet<Preview[]>(`/api/v1/apps/${appID}/previews`),
    enabled: !!appID,
  });
}
